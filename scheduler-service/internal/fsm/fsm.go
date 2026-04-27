package fsm

import (
	"fmt"
	"sync"
)

// State 任务状态
type State int8

const (
	StatePending State = iota
	StateRunning
	StateDone
	StateFailed
	StateRetry
)

func (s State) String() string {
	return [...]string{"pending", "running", "done", "failed", "retry"}[s]
}

// Event 触发状态转移的事件
type Event int8

const (
	EventStart   Event = iota // Pending  → Running
	EventSucceed              // Running  → Done
	EventFail                 // Running  → Failed
	EventRetry                // Failed   → Retry
	EventRetryStart           // Retry    → Running
	EventCancel               // Pending  → Failed
)

// transition 定义合法的状态转移
// map[当前状态][事件] = 目标状态
var transitions = map[State]map[Event]State{
	StatePending: {
		EventStart:  StateRunning,
		EventCancel: StateFailed,
	},
	StateRunning: {
		EventSucceed: StateDone,
		EventFail:    StateFailed,
	},
	StateFailed: {
		EventRetry: StateRetry,
	},
	StateRetry: {
		EventRetryStart: StateRunning,
	},
}

// TaskFSM 任务状态机
// ─────────────────────────────────────────────────────────
// 面试考点：为什么要用状态机？
//   → 防止非法状态转移（如 Done 任务被重复执行）
//   → 状态变更有迹可查，方便排查问题
//   → 并发安全（mutex 保护）
type TaskFSM struct {
	mu       sync.Mutex
	state    State
	taskID   string
	onChange func(taskID string, from, to State) // 状态变更回调
}

func New(taskID string, initial State, onChange func(string, State, State)) *TaskFSM {
	return &TaskFSM{
		state:    initial,
		taskID:   taskID,
		onChange: onChange,
	}
}

// Trigger 触发事件，执行状态转移
func (f *TaskFSM) Trigger(event Event) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	events, ok := transitions[f.state]
	if !ok {
		return fmt.Errorf("no transitions from state %s", f.state)
	}

	next, ok := events[event]
	if !ok {
		return fmt.Errorf("invalid transition: state=%s event=%d", f.state, event)
	}

	from := f.state
	f.state = next

	if f.onChange != nil {
		go f.onChange(f.taskID, from, next)
	}

	return nil
}

func (f *TaskFSM) State() State {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state
}

func (f *TaskFSM) Is(s State) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.state == s
}
