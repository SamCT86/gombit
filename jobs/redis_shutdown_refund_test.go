package jobs_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gombit-dev/gombit/jobs"
)

// redisPauseAfterReserve preserves the real Redis receipt but delays its
// return to the worker until shutdown has canceled the reserve context.
type redisPauseAfterReserve struct {
	jobs.Queue
	entered chan struct{}
}

func (q *redisPauseAfterReserve) Reserve(ctx context.Context, queues []string, lease time.Duration) (jobs.Delivery, error) {
	d, err := q.Queue.Reserve(ctx, queues, lease)
	if err == nil {
		select {
		case q.entered <- struct{}{}:
		default:
		}
		<-ctx.Done()
	}
	return d, err
}

func TestRedisUnstartedShutdownRefundsAllAttempts(t *testing.T) {
	addr := redisTestAddr(t)
	namespace := testNamespace()
	realQueue := jobs.NewRedisQueue(redisClient(t, addr, namespace), namespace)

	reg := jobs.NewRegistry()
	var ran atomic.Bool
	jobs.MustRegister(reg, func(context.Context, blockJob) error {
		ran.Store(true)
		return nil
	})
	dispatchN(t, jobs.NewDispatcher(reg, realQueue), 1)

	for i := 0; i < 3; i++ {
		q := &redisPauseAfterReserve{Queue: realQueue, entered: make(chan struct{}, 1)}
		w, err := jobs.NewWorker(reg, q, jobs.WorkerOptions{
			Queues: []string{"default"}, PollInterval: 5 * time.Millisecond,
		})
		if err != nil {
			t.Fatal(err)
		}
		stop := startWorker(t, w)
		select {
		case <-q.entered:
		case <-time.After(3 * time.Second):
			t.Fatal("Redis worker never reserved a delivery")
		}
		if err := stop(); err != nil {
			t.Fatalf("shutdown #%d: %v", i+1, err)
		}
	}
	if ran.Load() {
		t.Fatal("handler ran despite shutdown before reserve reply")
	}
	next, err := realQueue.Reserve(context.Background(), []string{"default"}, time.Minute)
	if err != nil || next.Envelope.Attempt != 1 || next.LeaseExpired {
		t.Fatalf("after three Redis-backed unstarted shutdowns: delivery %+v, err %v; want attempt=1 and no lease-expiry reclaim", next, err)
	}
	if err := realQueue.Ack(context.Background(), next); err != nil {
		t.Fatal(err)
	}
}
