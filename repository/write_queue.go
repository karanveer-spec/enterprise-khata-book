package repository

import (
	"database/sql"
	"fmt"
	"sync"
)

// WriteTask represents an atomic unit of write work passed to the write queue.
type WriteTask struct {
	Execute func(tx *sql.Tx) error
	ErrChan chan error
}

// WriteQueue serializes all write operations (INSERT, UPDATE, DELETE) through a single worker goroutine.
// This completely prevents SQLite "database is locked" errors under concurrent multi-user load,
// while allowing concurrent reads (SELECT) to execute without blocking via SQLite WAL mode.
type WriteQueue struct {
	db       *sql.DB
	taskChan chan WriteTask
	quit     chan struct{}
	wg       sync.WaitGroup
}

// NewWriteQueue initializes and starts the channel-based write queue.
func NewWriteQueue(db *sql.DB, bufferSize int) *WriteQueue {
	if bufferSize <= 0 {
		bufferSize = 256
	}
	wq := &WriteQueue{
		db:       db,
		taskChan: make(chan WriteTask, bufferSize),
		quit:     make(chan struct{}),
	}
	wq.wg.Add(1)
	go wq.worker()
	return wq
}

func (wq *WriteQueue) worker() {
	defer wq.wg.Done()
	for {
		select {
		case <-wq.quit:
			// Drain remaining tasks before shutting down
			for len(wq.taskChan) > 0 {
				task := <-wq.taskChan
				wq.executeTask(task)
			}
			return
		case task, ok := <-wq.taskChan:
			if !ok {
				return
			}
			wq.executeTask(task)
		}
	}
}

func (wq *WriteQueue) executeTask(task WriteTask) {
	tx, err := wq.db.Begin()
	if err != nil {
		task.ErrChan <- fmt.Errorf("write queue failed to begin transaction: %w", err)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			task.ErrChan <- fmt.Errorf("write task panicked: %v", r)
		}
	}()

	if err := task.Execute(tx); err != nil {
		_ = tx.Rollback()
		task.ErrChan <- err
		return
	}

	if err := tx.Commit(); err != nil {
		task.ErrChan <- fmt.Errorf("write queue failed to commit transaction: %w", err)
		return
	}

	task.ErrChan <- nil
}

// ExecWrite executes a write operation inside a serialized transaction worker.
func (wq *WriteQueue) ExecWrite(fn func(tx *sql.Tx) error) error {
	errChan := make(chan error, 1)
	wq.taskChan <- WriteTask{
		Execute: fn,
		ErrChan: errChan,
	}
	return <-errChan
}

// Close gracefully flushes pending write tasks and stops the worker.
func (wq *WriteQueue) Close() {
	close(wq.quit)
	wq.wg.Wait()
}
