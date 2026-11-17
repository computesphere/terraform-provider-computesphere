package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	csv2 "github.com/computesphere/computesphere-go"
)

// After a delete returns, the API keeps the deleted service's name for a
// few seconds, and a create with that name in that environment answers 409
// until the delete finishes. A replacement under the same name (a change
// that destroys and creates the resource) sends its create right after the
// delete, so that create waits the hold out.
//
// Only a name this provider process just deleted is waited for: any other
// 409 is a real conflict and fails at once.

// deletedName is a service name deleted in an environment, per client.
type deletedName struct {
	client    *csv2.ClientWithResponses
	env, name string
}

// recentDeletes maps each deletedName to the time its delete returned.
var recentDeletes sync.Map

// deleteMemory is how long a deleted name is remembered for a replacement.
const deleteMemory = 10 * time.Minute

// The hold is waited out with backoff: nameRetryStart, doubling up to
// nameRetryCap between attempts, for at most nameRetryMax.
var (
	nameRetryStart = 500 * time.Millisecond
	nameRetryCap   = 5 * time.Second
	nameRetryMax   = 2 * time.Minute
)

// SetNameRetryForTest shortens the wait for a deleted name. Test use only.
func SetNameRetryForTest(start, maxWait time.Duration) {
	nameRetryStart, nameRetryCap, nameRetryMax = start, start, maxWait
}

func noteDeleted(c *csv2.ClientWithResponses, env, name string) {
	if env != "" && name != "" {
		recentDeletes.Store(deletedName{c, env, name}, time.Now())
	}
}

// justDeleted reports whether this process deleted the name in the
// environment within deleteMemory.
func justDeleted(c *csv2.ClientWithResponses, env, name string) bool {
	if env == "" || name == "" {
		return false
	}
	v, ok := recentDeletes.Load(deletedName{c, env, name})
	if !ok {
		return false
	}
	at, ok := v.(time.Time)
	return ok && time.Since(at) < deleteMemory
}

// isConflict reports whether a response is the API's 409 conflict problem.
func isConflict(status int, body []byte) bool {
	if status != http.StatusConflict {
		return false
	}
	var p struct {
		Code string `json:"code"`
	}
	return json.Unmarshal(body, &p) == nil && p.Code == "conflict"
}

// createWaitingForName sends a create, and while the name it reuses is
// still held by a delete this process just made, sends it again with
// backoff. The last answer is returned as is.
func createWaitingForName(ctx context.Context, held bool, send func() (status int, body []byte, err error)) error {
	wait, deadline := nameRetryStart, time.Now().Add(nameRetryMax)
	for {
		status, body, err := send()
		if err != nil || !held || !isConflict(status, body) {
			return err
		}
		left := time.Until(deadline)
		if left <= 0 {
			return nil
		}
		select {
		case <-ctx.Done():
			return nil // the 409 stands; the caller reports it
		case <-time.After(min(wait, left)):
		}
		wait = min(wait*2, nameRetryCap)
	}
}
