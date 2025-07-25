// Code generated by http://github.com/gojuno/minimock (v3.4.3). DO NOT EDIT.

package server

import (
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
)

// EncoderMock implements Encoder
type EncoderMock struct {
	t          minimock.Tester
	finishOnce sync.Once

	funcEncode          func(v any) (err error)
	funcEncodeOrigin    string
	inspectFuncEncode   func(v any)
	afterEncodeCounter  uint64
	beforeEncodeCounter uint64
	EncodeMock          mEncoderMockEncode
}

// NewEncoderMock returns a mock for Encoder
func NewEncoderMock(t minimock.Tester) *EncoderMock {
	m := &EncoderMock{t: t}

	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.EncodeMock = mEncoderMockEncode{mock: m}
	m.EncodeMock.callArgs = []*EncoderMockEncodeParams{}

	t.Cleanup(m.MinimockFinish)

	return m
}

type mEncoderMockEncode struct {
	optional           bool
	mock               *EncoderMock
	defaultExpectation *EncoderMockEncodeExpectation
	expectations       []*EncoderMockEncodeExpectation

	callArgs []*EncoderMockEncodeParams
	mutex    sync.RWMutex

	expectedInvocations       uint64
	expectedInvocationsOrigin string
}

// EncoderMockEncodeExpectation specifies expectation struct of the Encoder.Encode
type EncoderMockEncodeExpectation struct {
	mock               *EncoderMock
	params             *EncoderMockEncodeParams
	paramPtrs          *EncoderMockEncodeParamPtrs
	expectationOrigins EncoderMockEncodeExpectationOrigins
	results            *EncoderMockEncodeResults
	returnOrigin       string
	Counter            uint64
}

// EncoderMockEncodeParams contains parameters of the Encoder.Encode
type EncoderMockEncodeParams struct {
	v any
}

// EncoderMockEncodeParamPtrs contains pointers to parameters of the Encoder.Encode
type EncoderMockEncodeParamPtrs struct {
	v *any
}

// EncoderMockEncodeResults contains results of the Encoder.Encode
type EncoderMockEncodeResults struct {
	err error
}

// EncoderMockEncodeOrigins contains origins of expectations of the Encoder.Encode
type EncoderMockEncodeExpectationOrigins struct {
	origin  string
	originV string
}

// Marks this method to be optional. The default behavior of any method with Return() is '1 or more', meaning
// the test will fail minimock's automatic final call check if the mocked method was not called at least once.
// Optional() makes method check to work in '0 or more' mode.
// It is NOT RECOMMENDED to use this option unless you really need it, as default behaviour helps to
// catch the problems when the expected method call is totally skipped during test run.
func (mmEncode *mEncoderMockEncode) Optional() *mEncoderMockEncode {
	mmEncode.optional = true
	return mmEncode
}

// Expect sets up expected params for Encoder.Encode
func (mmEncode *mEncoderMockEncode) Expect(v any) *mEncoderMockEncode {
	if mmEncode.mock.funcEncode != nil {
		mmEncode.mock.t.Fatalf("EncoderMock.Encode mock is already set by Set")
	}

	if mmEncode.defaultExpectation == nil {
		mmEncode.defaultExpectation = &EncoderMockEncodeExpectation{}
	}

	if mmEncode.defaultExpectation.paramPtrs != nil {
		mmEncode.mock.t.Fatalf("EncoderMock.Encode mock is already set by ExpectParams functions")
	}

	mmEncode.defaultExpectation.params = &EncoderMockEncodeParams{v}
	mmEncode.defaultExpectation.expectationOrigins.origin = minimock.CallerInfo(1)
	for _, e := range mmEncode.expectations {
		if minimock.Equal(e.params, mmEncode.defaultExpectation.params) {
			mmEncode.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmEncode.defaultExpectation.params)
		}
	}

	return mmEncode
}

// ExpectVParam1 sets up expected param v for Encoder.Encode
func (mmEncode *mEncoderMockEncode) ExpectVParam1(v any) *mEncoderMockEncode {
	if mmEncode.mock.funcEncode != nil {
		mmEncode.mock.t.Fatalf("EncoderMock.Encode mock is already set by Set")
	}

	if mmEncode.defaultExpectation == nil {
		mmEncode.defaultExpectation = &EncoderMockEncodeExpectation{}
	}

	if mmEncode.defaultExpectation.params != nil {
		mmEncode.mock.t.Fatalf("EncoderMock.Encode mock is already set by Expect")
	}

	if mmEncode.defaultExpectation.paramPtrs == nil {
		mmEncode.defaultExpectation.paramPtrs = &EncoderMockEncodeParamPtrs{}
	}
	mmEncode.defaultExpectation.paramPtrs.v = &v
	mmEncode.defaultExpectation.expectationOrigins.originV = minimock.CallerInfo(1)

	return mmEncode
}

// Inspect accepts an inspector function that has same arguments as the Encoder.Encode
func (mmEncode *mEncoderMockEncode) Inspect(f func(v any)) *mEncoderMockEncode {
	if mmEncode.mock.inspectFuncEncode != nil {
		mmEncode.mock.t.Fatalf("Inspect function is already set for EncoderMock.Encode")
	}

	mmEncode.mock.inspectFuncEncode = f

	return mmEncode
}

// Return sets up results that will be returned by Encoder.Encode
func (mmEncode *mEncoderMockEncode) Return(err error) *EncoderMock {
	if mmEncode.mock.funcEncode != nil {
		mmEncode.mock.t.Fatalf("EncoderMock.Encode mock is already set by Set")
	}

	if mmEncode.defaultExpectation == nil {
		mmEncode.defaultExpectation = &EncoderMockEncodeExpectation{mock: mmEncode.mock}
	}
	mmEncode.defaultExpectation.results = &EncoderMockEncodeResults{err}
	mmEncode.defaultExpectation.returnOrigin = minimock.CallerInfo(1)
	return mmEncode.mock
}

// Set uses given function f to mock the Encoder.Encode method
func (mmEncode *mEncoderMockEncode) Set(f func(v any) (err error)) *EncoderMock {
	if mmEncode.defaultExpectation != nil {
		mmEncode.mock.t.Fatalf("Default expectation is already set for the Encoder.Encode method")
	}

	if len(mmEncode.expectations) > 0 {
		mmEncode.mock.t.Fatalf("Some expectations are already set for the Encoder.Encode method")
	}

	mmEncode.mock.funcEncode = f
	mmEncode.mock.funcEncodeOrigin = minimock.CallerInfo(1)
	return mmEncode.mock
}

// When sets expectation for the Encoder.Encode which will trigger the result defined by the following
// Then helper
func (mmEncode *mEncoderMockEncode) When(v any) *EncoderMockEncodeExpectation {
	if mmEncode.mock.funcEncode != nil {
		mmEncode.mock.t.Fatalf("EncoderMock.Encode mock is already set by Set")
	}

	expectation := &EncoderMockEncodeExpectation{
		mock:               mmEncode.mock,
		params:             &EncoderMockEncodeParams{v},
		expectationOrigins: EncoderMockEncodeExpectationOrigins{origin: minimock.CallerInfo(1)},
	}
	mmEncode.expectations = append(mmEncode.expectations, expectation)
	return expectation
}

// Then sets up Encoder.Encode return parameters for the expectation previously defined by the When method
func (e *EncoderMockEncodeExpectation) Then(err error) *EncoderMock {
	e.results = &EncoderMockEncodeResults{err}
	return e.mock
}

// Times sets number of times Encoder.Encode should be invoked
func (mmEncode *mEncoderMockEncode) Times(n uint64) *mEncoderMockEncode {
	if n == 0 {
		mmEncode.mock.t.Fatalf("Times of EncoderMock.Encode mock can not be zero")
	}
	mm_atomic.StoreUint64(&mmEncode.expectedInvocations, n)
	mmEncode.expectedInvocationsOrigin = minimock.CallerInfo(1)
	return mmEncode
}

func (mmEncode *mEncoderMockEncode) invocationsDone() bool {
	if len(mmEncode.expectations) == 0 && mmEncode.defaultExpectation == nil && mmEncode.mock.funcEncode == nil {
		return true
	}

	totalInvocations := mm_atomic.LoadUint64(&mmEncode.mock.afterEncodeCounter)
	expectedInvocations := mm_atomic.LoadUint64(&mmEncode.expectedInvocations)

	return totalInvocations > 0 && (expectedInvocations == 0 || expectedInvocations == totalInvocations)
}

// Encode implements Encoder
func (mmEncode *EncoderMock) Encode(v any) (err error) {
	mm_atomic.AddUint64(&mmEncode.beforeEncodeCounter, 1)
	defer mm_atomic.AddUint64(&mmEncode.afterEncodeCounter, 1)

	mmEncode.t.Helper()

	if mmEncode.inspectFuncEncode != nil {
		mmEncode.inspectFuncEncode(v)
	}

	mm_params := EncoderMockEncodeParams{v}

	// Record call args
	mmEncode.EncodeMock.mutex.Lock()
	mmEncode.EncodeMock.callArgs = append(mmEncode.EncodeMock.callArgs, &mm_params)
	mmEncode.EncodeMock.mutex.Unlock()

	for _, e := range mmEncode.EncodeMock.expectations {
		if minimock.Equal(*e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.err
		}
	}

	if mmEncode.EncodeMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmEncode.EncodeMock.defaultExpectation.Counter, 1)
		mm_want := mmEncode.EncodeMock.defaultExpectation.params
		mm_want_ptrs := mmEncode.EncodeMock.defaultExpectation.paramPtrs

		mm_got := EncoderMockEncodeParams{v}

		if mm_want_ptrs != nil {

			if mm_want_ptrs.v != nil && !minimock.Equal(*mm_want_ptrs.v, mm_got.v) {
				mmEncode.t.Errorf("EncoderMock.Encode got unexpected parameter v, expected at\n%s:\nwant: %#v\n got: %#v%s\n",
					mmEncode.EncodeMock.defaultExpectation.expectationOrigins.originV, *mm_want_ptrs.v, mm_got.v, minimock.Diff(*mm_want_ptrs.v, mm_got.v))
			}

		} else if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmEncode.t.Errorf("EncoderMock.Encode got unexpected parameters, expected at\n%s:\nwant: %#v\n got: %#v%s\n",
				mmEncode.EncodeMock.defaultExpectation.expectationOrigins.origin, *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmEncode.EncodeMock.defaultExpectation.results
		if mm_results == nil {
			mmEncode.t.Fatal("No results are set for the EncoderMock.Encode")
		}
		return (*mm_results).err
	}
	if mmEncode.funcEncode != nil {
		return mmEncode.funcEncode(v)
	}
	mmEncode.t.Fatalf("Unexpected call to EncoderMock.Encode. %v", v)
	return
}

// EncodeAfterCounter returns a count of finished EncoderMock.Encode invocations
func (mmEncode *EncoderMock) EncodeAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmEncode.afterEncodeCounter)
}

// EncodeBeforeCounter returns a count of EncoderMock.Encode invocations
func (mmEncode *EncoderMock) EncodeBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmEncode.beforeEncodeCounter)
}

// Calls returns a list of arguments used in each call to EncoderMock.Encode.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmEncode *mEncoderMockEncode) Calls() []*EncoderMockEncodeParams {
	mmEncode.mutex.RLock()

	argCopy := make([]*EncoderMockEncodeParams, len(mmEncode.callArgs))
	copy(argCopy, mmEncode.callArgs)

	mmEncode.mutex.RUnlock()

	return argCopy
}

// MinimockEncodeDone returns true if the count of the Encode invocations corresponds
// the number of defined expectations
func (m *EncoderMock) MinimockEncodeDone() bool {
	if m.EncodeMock.optional {
		// Optional methods provide '0 or more' call count restriction.
		return true
	}

	for _, e := range m.EncodeMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	return m.EncodeMock.invocationsDone()
}

// MinimockEncodeInspect logs each unmet expectation
func (m *EncoderMock) MinimockEncodeInspect() {
	for _, e := range m.EncodeMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to EncoderMock.Encode at\n%s with params: %#v", e.expectationOrigins.origin, *e.params)
		}
	}

	afterEncodeCounter := mm_atomic.LoadUint64(&m.afterEncodeCounter)
	// if default expectation was set then invocations count should be greater than zero
	if m.EncodeMock.defaultExpectation != nil && afterEncodeCounter < 1 {
		if m.EncodeMock.defaultExpectation.params == nil {
			m.t.Errorf("Expected call to EncoderMock.Encode at\n%s", m.EncodeMock.defaultExpectation.returnOrigin)
		} else {
			m.t.Errorf("Expected call to EncoderMock.Encode at\n%s with params: %#v", m.EncodeMock.defaultExpectation.expectationOrigins.origin, *m.EncodeMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcEncode != nil && afterEncodeCounter < 1 {
		m.t.Errorf("Expected call to EncoderMock.Encode at\n%s", m.funcEncodeOrigin)
	}

	if !m.EncodeMock.invocationsDone() && afterEncodeCounter > 0 {
		m.t.Errorf("Expected %d calls to EncoderMock.Encode at\n%s but found %d calls",
			mm_atomic.LoadUint64(&m.EncodeMock.expectedInvocations), m.EncodeMock.expectedInvocationsOrigin, afterEncodeCounter)
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *EncoderMock) MinimockFinish() {
	m.finishOnce.Do(func() {
		if !m.minimockDone() {
			m.MinimockEncodeInspect()
		}
	})
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *EncoderMock) MinimockWait(timeout mm_time.Duration) {
	timeoutCh := mm_time.After(timeout)
	for {
		if m.minimockDone() {
			return
		}
		select {
		case <-timeoutCh:
			m.MinimockFinish()
			return
		case <-mm_time.After(10 * mm_time.Millisecond):
		}
	}
}

func (m *EncoderMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockEncodeDone()
}
