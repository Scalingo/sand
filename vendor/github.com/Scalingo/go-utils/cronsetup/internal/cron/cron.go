package cron

import (
	"context"
	"fmt"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"
	"time"

	"github.com/Scalingo/go-utils/errors/v3"
	"github.com/Scalingo/go-utils/logger"
)

// Cron keeps track of any number of entries, invoking the associated func as
// specified by the schedule. It may be started, stopped, and the entries may
// be inspected while running.
type Cron struct {
	entries           []*Entry
	stop              chan struct{}
	add               chan *Entry
	snapshot          chan []*Entry
	etcdErrorsHandler func(context.Context, Job, error)
	errorsHandler     func(context.Context, Job, error)
	funcCtx           func(context.Context, Job) context.Context
	running           bool
	etcdMutexBuilder  EtcdMutexBuilder
}

// Job contains 3 mandatory options to define a job
type Job struct {
	// Name of the job
	Name string
	// Cron-formatted rhythm (ie. 0,10,30 1-5 0 * * *)
	Rhythm string
	// Routine method
	Func func(context.Context) error
}

func (j Job) Run(ctx context.Context) error {
	return j.Func(ctx)
}

var (
	nonAlphaNumerical = regexp.MustCompile("[^a-z0-9_]")
)

func (j Job) canonicalName() string {
	jobNameLowerCase := strings.ToLower(j.Name)
	// Replace non alphanumeric characters with '_'
	jobNameWithoutSpecialCharacters := nonAlphaNumerical.ReplaceAllString(
		jobNameLowerCase, "_",
	)
	return toSnakeCase(jobNameWithoutSpecialCharacters)
}

// toSnakeCase converts the string s to a snake case:
//   - all spaces are replaced with a _
//   - non-alphanumeric characters are replaced with a _
//
// Credits goes to https://github.com/iancoleman/strcase under the MIT license for this code.
func toSnakeCase(s string) string {
	delimiter := byte('_')
	s = strings.TrimSpace(s)
	n := strings.Builder{}
	n.Grow(len(s) + 2) // nominal 2 bytes of extra space for inserted delimiters
	for i, v := range []byte(s) {
		vIsCap := v >= 'A' && v <= 'Z'
		vIsLow := v >= 'a' && v <= 'z'
		if vIsCap {
			v += 'a'
			v -= 'A'
		}

		// treat acronyms as words, eg for JSONData -> JSON is a whole word
		//nolint:nestif
		if i+1 < len(s) {
			next := s[i+1]
			vIsNum := v >= '0' && v <= '9'
			nextIsCap := next >= 'A' && next <= 'Z'
			nextIsLow := next >= 'a' && next <= 'z'
			nextIsNum := next >= '0' && next <= '9'
			// add underscore if next letter case type is changed
			if (vIsCap && (nextIsLow || nextIsNum)) || (vIsLow && (nextIsCap || nextIsNum)) || (vIsNum && (nextIsCap || nextIsLow)) {
				if vIsCap && nextIsLow {
					if prevIsCap := i > 0 && s[i-1] >= 'A' && s[i-1] <= 'Z'; prevIsCap {
						n.WriteByte(delimiter)
					}
				}
				n.WriteByte(v)
				if vIsLow || vIsNum || nextIsNum {
					n.WriteByte(delimiter)
				}
				continue
			}
		}

		if v == ' ' || v == '_' || v == '-' || v == '.' {
			n.WriteByte(delimiter)
		} else {
			n.WriteByte(v)
		}
	}

	return n.String()
}

// The Schedule describes a job's duty cycle.
type Schedule interface {
	// Return the next activation time, later than the given time.
	// Next is invoked initially, and then each time the job is run.
	Next(t time.Time) time.Time
}

// Entry consists of a schedule and the func to execute on that schedule.
type Entry struct {
	// The schedule on which this job should be run.
	Schedule Schedule

	// The next time the job will run. This is the zero time if Cron has not been
	// started or this entry's schedule is unsatisfiable
	Next time.Time

	// The last time this job was run. This is the zero time if the job has never
	// been run.
	Prev time.Time

	// The Job o run.
	Job Job
}

// byTime is a wrapper for sorting the entry array by time
// (with zero time at the end).
type byTime []*Entry

func (s byTime) Len() int      { return len(s) }
func (s byTime) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s byTime) Less(i, j int) bool {
	// Two zero times should return false.
	// Otherwise, zero is "greater" than any other time.
	// (To sort it at the end of the list.)
	if s[i].Next.IsZero() {
		return false
	}
	if s[j].Next.IsZero() {
		return true
	}
	return s[i].Next.Before(s[j].Next)
}

type Opt func(cron *Cron)

// WithEtcdErrorsHandler updates the default etcd error handler. It is called when an error occurs while interacting with etcd.
// The default handler outputs a log line on stdout.
func WithEtcdErrorsHandler(f func(context.Context, Job, error)) Opt {
	return Opt(func(cron *Cron) {
		cron.etcdErrorsHandler = f
	})
}

// WithErrorsHandler updates the default error handler. It is called when an error occurs while executing a cron job.
// The default handler outputs a log line on stdout.
func WithErrorsHandler(f func(context.Context, Job, error)) Opt {
	return Opt(func(cron *Cron) {
		cron.errorsHandler = f
	})
}

// WithEtcdMutexBuilder sets an etcd client to pose mutex. Setting such a client enables the distributed mode.
func WithEtcdMutexBuilder(etcdMutexBuilder EtcdMutexBuilder) Opt {
	return Opt(func(cron *Cron) {
		cron.etcdMutexBuilder = etcdMutexBuilder
	})
}

// WithFuncCtx is a callback executed at the beginning of the execution of each entry. It only returns a context.
func WithFuncCtx(f func(context.Context, Job) context.Context) Opt {
	return Opt(func(cron *Cron) {
		cron.funcCtx = f
	})
}

// New returns a new cron job runner.
func New(opts ...Opt) (*Cron, error) {
	cron := &Cron{
		entries:  nil,
		add:      make(chan *Entry),
		stop:     make(chan struct{}),
		snapshot: make(chan []*Entry),
		running:  false,
	}
	for _, opt := range opts {
		opt(cron)
	}

	if cron.etcdErrorsHandler == nil {
		cron.etcdErrorsHandler = func(ctx context.Context, j Job, err error) {
			_, log := logger.WithFieldToCtx(ctx, "job_name", j.Name)
			log.WithError(err).Infof("[cron] etcd error when handling '%v' job: %v", j.Name, err)
		}
	}

	if cron.errorsHandler == nil {
		cron.errorsHandler = func(ctx context.Context, j Job, err error) {
			_, log := logger.WithFieldToCtx(ctx, "job_name", j.Name)
			log.WithError(err).Infof("[cron] error when handling '%v' job: %v", j.Name, err)
		}
	}

	return cron, nil
}

// AddFunc adds a Job to the Cron to be run on the given schedule.
func (c *Cron) AddJob(job Job) error {
	schedule, err := Parse(job.Rhythm)
	if err != nil {
		return err
	}
	c.Schedule(schedule, job)
	return nil
}

// Schedule adds a Job to the Cron to be run on the given schedule.
func (c *Cron) Schedule(schedule Schedule, job Job) {
	entry := &Entry{
		Schedule: schedule,
		Job:      job,
	}
	if !c.running {
		c.entries = append(c.entries, entry)
		return
	}

	c.add <- entry
}

// Entries returns a snapshot of the cron entries.
func (c *Cron) Entries() []*Entry {
	if c.running {
		c.snapshot <- nil
		x := <-c.snapshot
		return x
	}
	return c.entrySnapshot()
}

// Start the cron scheduler in its own go-routine.
func (c *Cron) Start(ctx context.Context) {
	c.running = true
	go c.run(ctx)
}

// Run the scheduler. This is private just due to the need to synchronize
// access to the 'running' state variable.
func (c *Cron) run(ctx context.Context) {
	// Figure out the next activation times for each entry.
	now := time.Now().Local()
	for _, entry := range c.entries {
		entry.Next = entry.Schedule.Next(now)
	}

	for {
		// Determine the next entry to run.
		sort.Sort(byTime(c.entries))

		var effective time.Time
		if len(c.entries) == 0 || c.entries[0].Next.IsZero() {
			// If there are no entries yet, just sleep - it still handles new entries
			// and stop requests.
			effective = now.AddDate(10, 0, 0)
		} else {
			effective = c.entries[0].Next
		}

		select {
		case now = <-time.After(effective.Sub(now)):
			// Run every entry whose next time was this effective time.
			for _, e := range c.entries {
				if e.Next != effective {
					break
				}
				e.Prev = e.Next
				e.Next = e.Schedule.Next(effective)

				go c.runEntry(ctx, effective, e)
			}
			continue

		case newEntry := <-c.add:
			c.entries = append(c.entries, newEntry)
			newEntry.Next = newEntry.Schedule.Next(now)

		case <-c.snapshot:
			c.snapshot <- c.entrySnapshot()

		case <-c.stop:
			return
		}

		// 'now' should be updated after newEntry and snapshot cases.
		now = time.Now().Local()
	}
}

// Stop the cron scheduler.
func (c *Cron) Stop() {
	c.stop <- struct{}{}
	c.running = false
}

// entrySnapshot returns a copy of the current cron entry list.
func (c *Cron) entrySnapshot() []*Entry {
	entries := []*Entry{}
	for _, e := range c.entries {
		entries = append(entries, &Entry{
			Schedule: e.Schedule,
			Next:     e.Next,
			Prev:     e.Prev,
			Job:      e.Job,
		})
	}
	return entries
}

func (c *Cron) runEntry(ctx context.Context, effective time.Time, e *Entry) {
	defer func() {
		r := recover()
		if r != nil {
			err, ok := r.(error)
			if !ok {
				err = fmt.Errorf("%v", r)
			}
			err = fmt.Errorf("panic: %v, stacktrace: %s", err, string(debug.Stack()))
			go c.errorsHandler(ctx, e.Job, err)
		}
	}()

	if c.funcCtx != nil {
		ctx = c.funcCtx(ctx, e.Job)
	}

	if c.etcdMutexBuilder == nil {
		// In the local mode, we execute the job anyway with no need of any mutex
		err := e.Job.Run(ctx)
		if err != nil {
			go c.errorsHandler(ctx, e.Job, err)
		}

		return
	}

	// In the distributed mode, we need to set a distributed mutex to ensure the job is only executed once.
	m, err := c.etcdMutexBuilder.NewMutex(fmt.Sprintf("etcd_cron/%s/%d", e.Job.canonicalName(), effective.Unix()))
	if err != nil {
		go c.etcdErrorsHandler(ctx, e.Job, errors.Wrapf(ctx, err, "create etcd mutex for job '%v'", e.Job.Name))
		return
	}
	lockCtx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	err = m.Lock(lockCtx)
	if err == context.DeadlineExceeded {
		return
	} else if err != nil {
		go c.etcdErrorsHandler(ctx, e.Job, errors.Wrapf(ctx, err, "lock mutex '%v'", m.Key()))
		return
	}

	err = e.Job.Run(ctx)
	if err != nil {
		go c.errorsHandler(ctx, e.Job, err)
		return
	}
}
