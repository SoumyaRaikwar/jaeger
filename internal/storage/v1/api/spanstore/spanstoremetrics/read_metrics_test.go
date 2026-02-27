// Copyright (c) 2019 The Jaeger Authors.
// Copyright (c) 2017 Uber Technologies, Inc.
// SPDX-License-Identifier: Apache-2.0

package spanstoremetrics_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/jaegertracing/jaeger-idl/model/v1"
	"github.com/jaegertracing/jaeger/internal/metricstest"
	"github.com/jaegertracing/jaeger/internal/storage/v1/api/spanstore"
	"github.com/jaegertracing/jaeger/internal/storage/v1/api/spanstore/spanstoremetrics"
)

type mockReader struct {
	mock.Mock
}

func (m *mockReader) GetTrace(ctx context.Context, params spanstore.GetTraceParameters) (*model.Trace, error) {
	args := m.Called(ctx, params)
	if trace := args.Get(0); trace != nil {
		return trace.(*model.Trace), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockReader) GetServices(ctx context.Context) ([]string, error) {
	args := m.Called(ctx)
	if services := args.Get(0); services != nil {
		return services.([]string), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockReader) GetOperations(ctx context.Context, params spanstore.OperationQueryParameters) ([]spanstore.Operation, error) {
	args := m.Called(ctx, params)
	if ops := args.Get(0); ops != nil {
		return ops.([]spanstore.Operation), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockReader) FindTraces(ctx context.Context, params *spanstore.TraceQueryParameters) ([]*model.Trace, error) {
	args := m.Called(ctx, params)
	if traces := args.Get(0); traces != nil {
		return traces.([]*model.Trace), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockReader) FindTraceIDs(ctx context.Context, params *spanstore.TraceQueryParameters) ([]model.TraceID, error) {
	args := m.Called(ctx, params)
	if ids := args.Get(0); ids != nil {
		return ids.([]model.TraceID), args.Error(1)
	}
	return nil, args.Error(1)
}

var _ spanstore.Reader = (*mockReader)(nil)

func TestSuccessfulUnderlyingCalls(t *testing.T) {
	mf := metricstest.NewFactory(0)

	mockReader := mockReader{}
	mrs := spanstoremetrics.NewReaderDecorator(&mockReader, mf)
	mockReader.On("GetServices", context.Background()).Return([]string{}, nil)
	mrs.GetServices(context.Background())
	operationQuery := spanstore.OperationQueryParameters{ServiceName: "something"}
	mockReader.On("GetOperations", context.Background(), operationQuery).
		Return([]spanstore.Operation{}, nil)
	mrs.GetOperations(context.Background(), operationQuery)
	mockReader.On("GetTrace", context.Background(), spanstore.GetTraceParameters{}).Return(&model.Trace{}, nil)
	mrs.GetTrace(context.Background(), spanstore.GetTraceParameters{})
	mockReader.On("FindTraces", context.Background(), &spanstore.TraceQueryParameters{}).
		Return([]*model.Trace{}, nil)
	mrs.FindTraces(context.Background(), &spanstore.TraceQueryParameters{})
	mockReader.On("FindTraceIDs", context.Background(), &spanstore.TraceQueryParameters{}).
		Return([]model.TraceID{}, nil)
	mrs.FindTraceIDs(context.Background(), &spanstore.TraceQueryParameters{})
	counters, gauges := mf.Snapshot()
	expecteds := map[string]int64{
		"requests|operation=get_operations|result=ok":  1,
		"requests|operation=get_operations|result=err": 0,
		"requests|operation=get_trace|result=ok":       1,
		"requests|operation=get_trace|result=err":      0,
		"requests|operation=find_traces|result=ok":     1,
		"requests|operation=find_traces|result=err":    0,
		"requests|operation=find_trace_ids|result=ok":  1,
		"requests|operation=find_trace_ids|result=err": 0,
		"requests|operation=get_services|result=ok":    1,
		"requests|operation=get_services|result=err":   0,
	}

	existingKeys := []string{
		"latency|operation=get_operations|result=ok.P50",
		"responses|operation=get_trace.P50",
		"latency|operation=find_traces|result=ok.P50", // this is not exhaustive
	}
	nonExistentKeys := []string{
		"latency|operation=get_operations|result=err.P50",
	}

	checkExpectedExistingAndNonExistentCounters(t, counters, expecteds, gauges, existingKeys, nonExistentKeys)
}

func checkExpectedExistingAndNonExistentCounters(t *testing.T,
	actualCounters,
	expectedCounters,
	actualGauges map[string]int64,
	existingKeys,
	nonExistentKeys []string,
) {
	for k, v := range expectedCounters {
		assert.Equal(t, v, actualCounters[k], k)
	}

	for _, k := range existingKeys {
		_, ok := actualGauges[k]
		assert.True(t, ok)
	}

	for _, k := range nonExistentKeys {
		_, ok := actualGauges[k]
		assert.False(t, ok)
	}
}

func TestFailingUnderlyingCalls(t *testing.T) {
	mf := metricstest.NewFactory(0)

	mockReader := mockReader{}
	mrs := spanstoremetrics.NewReaderDecorator(&mockReader, mf)
	mockReader.On("GetServices", context.Background()).
		Return(nil, errors.New("Failure"))
	mrs.GetServices(context.Background())
	operationQuery := spanstore.OperationQueryParameters{ServiceName: "something"}
	mockReader.On("GetOperations", context.Background(), operationQuery).
		Return(nil, errors.New("Failure"))
	mrs.GetOperations(context.Background(), operationQuery)
	mockReader.On("GetTrace", context.Background(), spanstore.GetTraceParameters{}).
		Return(nil, errors.New("Failure"))
	mrs.GetTrace(context.Background(), spanstore.GetTraceParameters{})
	mockReader.On("FindTraces", context.Background(), &spanstore.TraceQueryParameters{}).
		Return(nil, errors.New("Failure"))
	mrs.FindTraces(context.Background(), &spanstore.TraceQueryParameters{})
	mockReader.On("FindTraceIDs", context.Background(), &spanstore.TraceQueryParameters{}).
		Return(nil, errors.New("Failure"))
	mrs.FindTraceIDs(context.Background(), &spanstore.TraceQueryParameters{})
	counters, gauges := mf.Snapshot()
	expecteds := map[string]int64{
		"requests|operation=get_operations|result=ok":  0,
		"requests|operation=get_operations|result=err": 1,
		"requests|operation=get_trace|result=ok":       0,
		"requests|operation=get_trace|result=err":      1,
		"requests|operation=find_traces|result=ok":     0,
		"requests|operation=find_traces|result=err":    1,
		"requests|operation=find_trace_ids|result=ok":  0,
		"requests|operation=find_trace_ids|result=err": 1,
		"requests|operation=get_services|result=ok":    0,
		"requests|operation=get_services|result=err":   1,
	}

	existingKeys := []string{
		"latency|operation=get_operations|result=err.P50",
	}

	nonExistentKeys := []string{
		"latency|operation=get_operations|result=ok.P50",
		"responses|operation=get_trace.P50",
		"latency|operation=query|result=ok.P50", // this is not exhaustive
	}

	checkExpectedExistingAndNonExistentCounters(t, counters, expecteds, gauges, existingKeys, nonExistentKeys)
}
