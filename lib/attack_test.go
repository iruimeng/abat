package abat

import (
	//"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestAttackRate(t *testing.T) {
	return
	t.Parallel()

	hitNum := uint64(0)

	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddUint64(&hitNum, 1)
		}),
	)
	tgt := Target{Method: "GET", Url: server.URL}

	rate := uint64(10)

	AttackRate(Targets{tgt}, rate, 1*time.Second)

	hits := atomic.LoadUint64(&hitNum)

	if hits != rate {
		t.Fatalf("wrong hits num, want %s, real %d\n", rate, hitNum)
	}

}
