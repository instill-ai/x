// Code generated by http://github.com/gojuno/minimock (v3.4.3). DO NOT EDIT.

package server

import (
	"sync"
	mm_atomic "sync/atomic"
	mm_time "time"

	"github.com/gojuno/minimock/v3"
)

// DecoderMock implements Decoder
type DecoderMock struct {
	t          minimock.Tester
	finishOnce sync.Once

	funcDecode          func(v any) (err error)
	funcDecodeOrigin    string
	inspectFuncDecode   func(v any)
	afterDecodeCounter  uint64
	beforeDecodeCounter uint64
	DecodeMock          mDecoderMockDecode
}

// NewDecoderMock returns a mock for Decoder
func NewDecoderMock(t minimock.Tester) *DecoderMock {
	m := &DecoderMock{t: t}

	if controller, ok := t.(minimock.MockController); ok {
		controller.RegisterMocker(m)
	}

	m.DecodeMock = mDecoderMockDecode{mock: m}
	m.DecodeMock.callArgs = []*DecoderMockDecodeParams{}

	t.Cleanup(m.MinimockFinish)

	return m
}

type mDecoderMockDecode struct {
	optional           bool
	mock               *DecoderMock
	defaultExpectation *DecoderMockDecodeExpectation
	expectations       []*DecoderMockDecodeExpectation

	callArgs []*DecoderMockDecodeParams
	mutex    sync.RWMutex

	expectedInvocations       uint64
	expectedInvocationsOrigin string
}

// DecoderMockDecodeExpectation specifies expectation struct of the Decoder.Decode
type DecoderMockDecodeExpectation struct {
	mock               *DecoderMock
	params             *DecoderMockDecodeParams
	paramPtrs          *DecoderMockDecodeParamPtrs
	expectationOrigins DecoderMockDecodeExpectationOrigins
	results            *DecoderMockDecodeResults
	returnOrigin       string
	Counter            uint64
}

// DecoderMockDecodeParams contains parameters of the Decoder.Decode
type DecoderMockDecodeParams struct {
	v any
}

// DecoderMockDecodeParamPtrs contains pointers to parameters of the Decoder.Decode
type DecoderMockDecodeParamPtrs struct {
	v *any
}

// DecoderMockDecodeResults contains results of the Decoder.Decode
type DecoderMockDecodeResults struct {
	err error
}

// DecoderMockDecodeOrigins contains origins of expectations of the Decoder.Decode
type DecoderMockDecodeExpectationOrigins struct {
	origin  string
	originV string
}

// Marks this method to be optional. The default behavior of any method with Return() is '1 or more', meaning
// the test will fail minimock's automatic final call check if the mocked method was not called at least once.
// Optional() makes method check to work in '0 or more' mode.
// It is NOT RECOMMENDED to use this option unless you really need it, as default behaviour helps to
// catch the problems when the expected method call is totally skipped during test run.
func (mmDecode *mDecoderMockDecode) Optional() *mDecoderMockDecode {
	mmDecode.optional = true
	return mmDecode
}

// Expect sets up expected params for Decoder.Decode
func (mmDecode *mDecoderMockDecode) Expect(v any) *mDecoderMockDecode {
	if mmDecode.mock.funcDecode != nil {
		mmDecode.mock.t.Fatalf("DecoderMock.Decode mock is already set by Set")
	}

	if mmDecode.defaultExpectation == nil {
		mmDecode.defaultExpectation = &DecoderMockDecodeExpectation{}
	}

	if mmDecode.defaultExpectation.paramPtrs != nil {
		mmDecode.mock.t.Fatalf("DecoderMock.Decode mock is already set by ExpectParams functions")
	}

	mmDecode.defaultExpectation.params = &DecoderMockDecodeParams{v}
	mmDecode.defaultExpectation.expectationOrigins.origin = minimock.CallerInfo(1)
	for _, e := range mmDecode.expectations {
		if minimock.Equal(e.params, mmDecode.defaultExpectation.params) {
			mmDecode.mock.t.Fatalf("Expectation set by When has same params: %#v", *mmDecode.defaultExpectation.params)
		}
	}

	return mmDecode
}

// ExpectVParam1 sets up expected param v for Decoder.Decode
func (mmDecode *mDecoderMockDecode) ExpectVParam1(v any) *mDecoderMockDecode {
	if mmDecode.mock.funcDecode != nil {
		mmDecode.mock.t.Fatalf("DecoderMock.Decode mock is already set by Set")
	}

	if mmDecode.defaultExpectation == nil {
		mmDecode.defaultExpectation = &DecoderMockDecodeExpectation{}
	}

	if mmDecode.defaultExpectation.params != nil {
		mmDecode.mock.t.Fatalf("DecoderMock.Decode mock is already set by Expect")
	}

	if mmDecode.defaultExpectation.paramPtrs == nil {
		mmDecode.defaultExpectation.paramPtrs = &DecoderMockDecodeParamPtrs{}
	}
	mmDecode.defaultExpectation.paramPtrs.v = &v
	mmDecode.defaultExpectation.expectationOrigins.originV = minimock.CallerInfo(1)

	return mmDecode
}

// Inspect accepts an inspector function that has same arguments as the Decoder.Decode
func (mmDecode *mDecoderMockDecode) Inspect(f func(v any)) *mDecoderMockDecode {
	if mmDecode.mock.inspectFuncDecode != nil {
		mmDecode.mock.t.Fatalf("Inspect function is already set for DecoderMock.Decode")
	}

	mmDecode.mock.inspectFuncDecode = f

	return mmDecode
}

// Return sets up results that will be returned by Decoder.Decode
func (mmDecode *mDecoderMockDecode) Return(err error) *DecoderMock {
	if mmDecode.mock.funcDecode != nil {
		mmDecode.mock.t.Fatalf("DecoderMock.Decode mock is already set by Set")
	}

	if mmDecode.defaultExpectation == nil {
		mmDecode.defaultExpectation = &DecoderMockDecodeExpectation{mock: mmDecode.mock}
	}
	mmDecode.defaultExpectation.results = &DecoderMockDecodeResults{err}
	mmDecode.defaultExpectation.returnOrigin = minimock.CallerInfo(1)
	return mmDecode.mock
}

// Set uses given function f to mock the Decoder.Decode method
func (mmDecode *mDecoderMockDecode) Set(f func(v any) (err error)) *DecoderMock {
	if mmDecode.defaultExpectation != nil {
		mmDecode.mock.t.Fatalf("Default expectation is already set for the Decoder.Decode method")
	}

	if len(mmDecode.expectations) > 0 {
		mmDecode.mock.t.Fatalf("Some expectations are already set for the Decoder.Decode method")
	}

	mmDecode.mock.funcDecode = f
	mmDecode.mock.funcDecodeOrigin = minimock.CallerInfo(1)
	return mmDecode.mock
}

// When sets expectation for the Decoder.Decode which will trigger the result defined by the following
// Then helper
func (mmDecode *mDecoderMockDecode) When(v any) *DecoderMockDecodeExpectation {
	if mmDecode.mock.funcDecode != nil {
		mmDecode.mock.t.Fatalf("DecoderMock.Decode mock is already set by Set")
	}

	expectation := &DecoderMockDecodeExpectation{
		mock:               mmDecode.mock,
		params:             &DecoderMockDecodeParams{v},
		expectationOrigins: DecoderMockDecodeExpectationOrigins{origin: minimock.CallerInfo(1)},
	}
	mmDecode.expectations = append(mmDecode.expectations, expectation)
	return expectation
}

// Then sets up Decoder.Decode return parameters for the expectation previously defined by the When method
func (e *DecoderMockDecodeExpectation) Then(err error) *DecoderMock {
	e.results = &DecoderMockDecodeResults{err}
	return e.mock
}

// Times sets number of times Decoder.Decode should be invoked
func (mmDecode *mDecoderMockDecode) Times(n uint64) *mDecoderMockDecode {
	if n == 0 {
		mmDecode.mock.t.Fatalf("Times of DecoderMock.Decode mock can not be zero")
	}
	mm_atomic.StoreUint64(&mmDecode.expectedInvocations, n)
	mmDecode.expectedInvocationsOrigin = minimock.CallerInfo(1)
	return mmDecode
}

func (mmDecode *mDecoderMockDecode) invocationsDone() bool {
	if len(mmDecode.expectations) == 0 && mmDecode.defaultExpectation == nil && mmDecode.mock.funcDecode == nil {
		return true
	}

	totalInvocations := mm_atomic.LoadUint64(&mmDecode.mock.afterDecodeCounter)
	expectedInvocations := mm_atomic.LoadUint64(&mmDecode.expectedInvocations)

	return totalInvocations > 0 && (expectedInvocations == 0 || expectedInvocations == totalInvocations)
}

// Decode implements Decoder
func (mmDecode *DecoderMock) Decode(v any) (err error) {
	mm_atomic.AddUint64(&mmDecode.beforeDecodeCounter, 1)
	defer mm_atomic.AddUint64(&mmDecode.afterDecodeCounter, 1)

	mmDecode.t.Helper()

	if mmDecode.inspectFuncDecode != nil {
		mmDecode.inspectFuncDecode(v)
	}

	mm_params := DecoderMockDecodeParams{v}

	// Record call args
	mmDecode.DecodeMock.mutex.Lock()
	mmDecode.DecodeMock.callArgs = append(mmDecode.DecodeMock.callArgs, &mm_params)
	mmDecode.DecodeMock.mutex.Unlock()

	for _, e := range mmDecode.DecodeMock.expectations {
		if minimock.Equal(*e.params, mm_params) {
			mm_atomic.AddUint64(&e.Counter, 1)
			return e.results.err
		}
	}

	if mmDecode.DecodeMock.defaultExpectation != nil {
		mm_atomic.AddUint64(&mmDecode.DecodeMock.defaultExpectation.Counter, 1)
		mm_want := mmDecode.DecodeMock.defaultExpectation.params
		mm_want_ptrs := mmDecode.DecodeMock.defaultExpectation.paramPtrs

		mm_got := DecoderMockDecodeParams{v}

		if mm_want_ptrs != nil {

			if mm_want_ptrs.v != nil && !minimock.Equal(*mm_want_ptrs.v, mm_got.v) {
				mmDecode.t.Errorf("DecoderMock.Decode got unexpected parameter v, expected at\n%s:\nwant: %#v\n got: %#v%s\n",
					mmDecode.DecodeMock.defaultExpectation.expectationOrigins.originV, *mm_want_ptrs.v, mm_got.v, minimock.Diff(*mm_want_ptrs.v, mm_got.v))
			}

		} else if mm_want != nil && !minimock.Equal(*mm_want, mm_got) {
			mmDecode.t.Errorf("DecoderMock.Decode got unexpected parameters, expected at\n%s:\nwant: %#v\n got: %#v%s\n",
				mmDecode.DecodeMock.defaultExpectation.expectationOrigins.origin, *mm_want, mm_got, minimock.Diff(*mm_want, mm_got))
		}

		mm_results := mmDecode.DecodeMock.defaultExpectation.results
		if mm_results == nil {
			mmDecode.t.Fatal("No results are set for the DecoderMock.Decode")
		}
		return (*mm_results).err
	}
	if mmDecode.funcDecode != nil {
		return mmDecode.funcDecode(v)
	}
	mmDecode.t.Fatalf("Unexpected call to DecoderMock.Decode. %v", v)
	return
}

// DecodeAfterCounter returns a count of finished DecoderMock.Decode invocations
func (mmDecode *DecoderMock) DecodeAfterCounter() uint64 {
	return mm_atomic.LoadUint64(&mmDecode.afterDecodeCounter)
}

// DecodeBeforeCounter returns a count of DecoderMock.Decode invocations
func (mmDecode *DecoderMock) DecodeBeforeCounter() uint64 {
	return mm_atomic.LoadUint64(&mmDecode.beforeDecodeCounter)
}

// Calls returns a list of arguments used in each call to DecoderMock.Decode.
// The list is in the same order as the calls were made (i.e. recent calls have a higher index)
func (mmDecode *mDecoderMockDecode) Calls() []*DecoderMockDecodeParams {
	mmDecode.mutex.RLock()

	argCopy := make([]*DecoderMockDecodeParams, len(mmDecode.callArgs))
	copy(argCopy, mmDecode.callArgs)

	mmDecode.mutex.RUnlock()

	return argCopy
}

// MinimockDecodeDone returns true if the count of the Decode invocations corresponds
// the number of defined expectations
func (m *DecoderMock) MinimockDecodeDone() bool {
	if m.DecodeMock.optional {
		// Optional methods provide '0 or more' call count restriction.
		return true
	}

	for _, e := range m.DecodeMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			return false
		}
	}

	return m.DecodeMock.invocationsDone()
}

// MinimockDecodeInspect logs each unmet expectation
func (m *DecoderMock) MinimockDecodeInspect() {
	for _, e := range m.DecodeMock.expectations {
		if mm_atomic.LoadUint64(&e.Counter) < 1 {
			m.t.Errorf("Expected call to DecoderMock.Decode at\n%s with params: %#v", e.expectationOrigins.origin, *e.params)
		}
	}

	afterDecodeCounter := mm_atomic.LoadUint64(&m.afterDecodeCounter)
	// if default expectation was set then invocations count should be greater than zero
	if m.DecodeMock.defaultExpectation != nil && afterDecodeCounter < 1 {
		if m.DecodeMock.defaultExpectation.params == nil {
			m.t.Errorf("Expected call to DecoderMock.Decode at\n%s", m.DecodeMock.defaultExpectation.returnOrigin)
		} else {
			m.t.Errorf("Expected call to DecoderMock.Decode at\n%s with params: %#v", m.DecodeMock.defaultExpectation.expectationOrigins.origin, *m.DecodeMock.defaultExpectation.params)
		}
	}
	// if func was set then invocations count should be greater than zero
	if m.funcDecode != nil && afterDecodeCounter < 1 {
		m.t.Errorf("Expected call to DecoderMock.Decode at\n%s", m.funcDecodeOrigin)
	}

	if !m.DecodeMock.invocationsDone() && afterDecodeCounter > 0 {
		m.t.Errorf("Expected %d calls to DecoderMock.Decode at\n%s but found %d calls",
			mm_atomic.LoadUint64(&m.DecodeMock.expectedInvocations), m.DecodeMock.expectedInvocationsOrigin, afterDecodeCounter)
	}
}

// MinimockFinish checks that all mocked methods have been called the expected number of times
func (m *DecoderMock) MinimockFinish() {
	m.finishOnce.Do(func() {
		if !m.minimockDone() {
			m.MinimockDecodeInspect()
		}
	})
}

// MinimockWait waits for all mocked methods to be called the expected number of times
func (m *DecoderMock) MinimockWait(timeout mm_time.Duration) {
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

func (m *DecoderMock) minimockDone() bool {
	done := true
	return done &&
		m.MinimockDecodeDone()
}
