package store

import (
    "context"
    "log"

    "github.com/FulgerX2007/grafana-scheduled-reports-app/pkg/model"
)

// writeOpType defines the type of write operation
type writeOpType int

const (
    opCreateSchedule writeOpType = iota
    opUpdateSchedule
    opDeleteSchedule
    opCreateRun
    opUpdateRun
    opUpsertSettings
)

// writeOp represents a single write operation with its response channel
type writeOp struct {
    opType   writeOpType
    data     interface{}
    response chan writeResult
}

// writeResult contains the result of a write operation
type writeResult struct {
    err error
    id  int64 // For operations that return an ID (Create operations)
}

// writeQueue manages serialized database writes
type writeQueue struct {
    queue  chan writeOp
    ctx    context.Context
    cancel context.CancelFunc
    done   chan struct{}
}

// newWriteQueue creates and starts a new write queue
func newWriteQueue(db *Store) *writeQueue {
    ctx, cancel := context.WithCancel(context.Background())
    wq := &writeQueue{
        queue:  make(chan writeOp, 100), // Buffer for 100 operations
        ctx:    ctx,
        cancel: cancel,
        done:   make(chan struct{}),
    }

    // Start the single writer goroutine
    go wq.processQueue(db)

    return wq
}

// processQueue is the single writer goroutine that processes all write operations sequentially
func (wq *writeQueue) processQueue(db *Store) {
    defer close(wq.done)

    for {
        select {
        case <-wq.ctx.Done():
            // Drain remaining operations before shutting down
            for {
                select {
                case op := <-wq.queue:
                    wq.executeOp(db, op)
                default:
                    log.Println("[WRITE QUEUE] Shutdown complete")
                    return
                }
            }

        case op := <-wq.queue:
            wq.executeOp(db, op)
        }
    }
}

// executeOp executes a single write operation
func (wq *writeQueue) executeOp(db *Store, op writeOp) {
    var result writeResult

    switch op.opType {
    case opCreateSchedule:
        schedule := op.data.(*model.Schedule)
        result.err = db.createScheduleDirect(schedule)
        result.id = schedule.ID

    case opUpdateSchedule:
        schedule := op.data.(*model.Schedule)
        result.err = db.updateScheduleDirect(schedule)

    case opDeleteSchedule:
        params := op.data.(deleteScheduleParams)
        result.err = db.deleteScheduleDirect(params.orgID, params.id)

    case opCreateRun:
        run := op.data.(*model.Run)
        result.err = db.createRunDirect(run)
        result.id = run.ID

    case opUpdateRun:
        run := op.data.(*model.Run)
        result.err = db.updateRunDirect(run)

    case opUpsertSettings:
        settings := op.data.(*model.Settings)
        result.err = db.upsertSettingsDirect(settings)
        result.id = settings.ID
    }

    // Send result back to caller
    op.response <- result
}

// enqueue adds a write operation to the queue and waits for the result
func (wq *writeQueue) enqueue(opType writeOpType, data interface{}) error {
    response := make(chan writeResult, 1)

    op := writeOp{
        opType:   opType,
        data:     data,
        response: response,
    }

    select {
    case wq.queue <- op:
        // Operation queued successfully
    case <-wq.ctx.Done():
        return wq.ctx.Err()
    }

    // Wait for result
    select {
    case result := <-response:
        return result.err
    case <-wq.ctx.Done():
        return wq.ctx.Err()
    }
}

// shutdown gracefully shuts down the write queue
func (wq *writeQueue) shutdown() {
    log.Println("[WRITE QUEUE] Shutting down...")
    wq.cancel()
    <-wq.done
}

// Helper structs for passing parameters
type deleteScheduleParams struct {
    orgID int64
    id    int64
}
