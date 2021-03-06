// Code generated by counterfeiter. DO NOT EDIT.
package fakes

import (
	"sync"

	"github.com/containernetworking/plugins/pkg/ns"
)

type NetNS struct {
	DoStub        func(toRun func(ns.NetNS) error) error
	doMutex       sync.RWMutex
	doArgsForCall []struct {
		toRun func(ns.NetNS) error
	}
	doReturns struct {
		result1 error
	}
	doReturnsOnCall map[int]struct {
		result1 error
	}
	SetStub        func() error
	setMutex       sync.RWMutex
	setArgsForCall []struct{}
	setReturns     struct {
		result1 error
	}
	setReturnsOnCall map[int]struct {
		result1 error
	}
	PathStub        func() string
	pathMutex       sync.RWMutex
	pathArgsForCall []struct{}
	pathReturns     struct {
		result1 string
	}
	pathReturnsOnCall map[int]struct {
		result1 string
	}
	FdStub        func() uintptr
	fdMutex       sync.RWMutex
	fdArgsForCall []struct{}
	fdReturns     struct {
		result1 uintptr
	}
	fdReturnsOnCall map[int]struct {
		result1 uintptr
	}
	CloseStub        func() error
	closeMutex       sync.RWMutex
	closeArgsForCall []struct{}
	closeReturns     struct {
		result1 error
	}
	closeReturnsOnCall map[int]struct {
		result1 error
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *NetNS) Do(toRun func(ns.NetNS) error) error {
	fake.doMutex.Lock()
	ret, specificReturn := fake.doReturnsOnCall[len(fake.doArgsForCall)]
	fake.doArgsForCall = append(fake.doArgsForCall, struct {
		toRun func(ns.NetNS) error
	}{toRun})
	fake.recordInvocation("Do", []interface{}{toRun})
	fake.doMutex.Unlock()
	if fake.DoStub != nil {
		return fake.DoStub(toRun)
	}
	if specificReturn {
		return ret.result1
	}
	return fake.doReturns.result1
}

func (fake *NetNS) DoCallCount() int {
	fake.doMutex.RLock()
	defer fake.doMutex.RUnlock()
	return len(fake.doArgsForCall)
}

func (fake *NetNS) DoArgsForCall(i int) func(ns.NetNS) error {
	fake.doMutex.RLock()
	defer fake.doMutex.RUnlock()
	return fake.doArgsForCall[i].toRun
}

func (fake *NetNS) DoReturns(result1 error) {
	fake.DoStub = nil
	fake.doReturns = struct {
		result1 error
	}{result1}
}

func (fake *NetNS) DoReturnsOnCall(i int, result1 error) {
	fake.DoStub = nil
	if fake.doReturnsOnCall == nil {
		fake.doReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.doReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *NetNS) Set() error {
	fake.setMutex.Lock()
	ret, specificReturn := fake.setReturnsOnCall[len(fake.setArgsForCall)]
	fake.setArgsForCall = append(fake.setArgsForCall, struct{}{})
	fake.recordInvocation("Set", []interface{}{})
	fake.setMutex.Unlock()
	if fake.SetStub != nil {
		return fake.SetStub()
	}
	if specificReturn {
		return ret.result1
	}
	return fake.setReturns.result1
}

func (fake *NetNS) SetCallCount() int {
	fake.setMutex.RLock()
	defer fake.setMutex.RUnlock()
	return len(fake.setArgsForCall)
}

func (fake *NetNS) SetReturns(result1 error) {
	fake.SetStub = nil
	fake.setReturns = struct {
		result1 error
	}{result1}
}

func (fake *NetNS) SetReturnsOnCall(i int, result1 error) {
	fake.SetStub = nil
	if fake.setReturnsOnCall == nil {
		fake.setReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.setReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *NetNS) Path() string {
	fake.pathMutex.Lock()
	ret, specificReturn := fake.pathReturnsOnCall[len(fake.pathArgsForCall)]
	fake.pathArgsForCall = append(fake.pathArgsForCall, struct{}{})
	fake.recordInvocation("Path", []interface{}{})
	fake.pathMutex.Unlock()
	if fake.PathStub != nil {
		return fake.PathStub()
	}
	if specificReturn {
		return ret.result1
	}
	return fake.pathReturns.result1
}

func (fake *NetNS) PathCallCount() int {
	fake.pathMutex.RLock()
	defer fake.pathMutex.RUnlock()
	return len(fake.pathArgsForCall)
}

func (fake *NetNS) PathReturns(result1 string) {
	fake.PathStub = nil
	fake.pathReturns = struct {
		result1 string
	}{result1}
}

func (fake *NetNS) PathReturnsOnCall(i int, result1 string) {
	fake.PathStub = nil
	if fake.pathReturnsOnCall == nil {
		fake.pathReturnsOnCall = make(map[int]struct {
			result1 string
		})
	}
	fake.pathReturnsOnCall[i] = struct {
		result1 string
	}{result1}
}

func (fake *NetNS) Fd() uintptr {
	fake.fdMutex.Lock()
	ret, specificReturn := fake.fdReturnsOnCall[len(fake.fdArgsForCall)]
	fake.fdArgsForCall = append(fake.fdArgsForCall, struct{}{})
	fake.recordInvocation("Fd", []interface{}{})
	fake.fdMutex.Unlock()
	if fake.FdStub != nil {
		return fake.FdStub()
	}
	if specificReturn {
		return ret.result1
	}
	return fake.fdReturns.result1
}

func (fake *NetNS) FdCallCount() int {
	fake.fdMutex.RLock()
	defer fake.fdMutex.RUnlock()
	return len(fake.fdArgsForCall)
}

func (fake *NetNS) FdReturns(result1 uintptr) {
	fake.FdStub = nil
	fake.fdReturns = struct {
		result1 uintptr
	}{result1}
}

func (fake *NetNS) FdReturnsOnCall(i int, result1 uintptr) {
	fake.FdStub = nil
	if fake.fdReturnsOnCall == nil {
		fake.fdReturnsOnCall = make(map[int]struct {
			result1 uintptr
		})
	}
	fake.fdReturnsOnCall[i] = struct {
		result1 uintptr
	}{result1}
}

func (fake *NetNS) Close() error {
	fake.closeMutex.Lock()
	ret, specificReturn := fake.closeReturnsOnCall[len(fake.closeArgsForCall)]
	fake.closeArgsForCall = append(fake.closeArgsForCall, struct{}{})
	fake.recordInvocation("Close", []interface{}{})
	fake.closeMutex.Unlock()
	if fake.CloseStub != nil {
		return fake.CloseStub()
	}
	if specificReturn {
		return ret.result1
	}
	return fake.closeReturns.result1
}

func (fake *NetNS) CloseCallCount() int {
	fake.closeMutex.RLock()
	defer fake.closeMutex.RUnlock()
	return len(fake.closeArgsForCall)
}

func (fake *NetNS) CloseReturns(result1 error) {
	fake.CloseStub = nil
	fake.closeReturns = struct {
		result1 error
	}{result1}
}

func (fake *NetNS) CloseReturnsOnCall(i int, result1 error) {
	fake.CloseStub = nil
	if fake.closeReturnsOnCall == nil {
		fake.closeReturnsOnCall = make(map[int]struct {
			result1 error
		})
	}
	fake.closeReturnsOnCall[i] = struct {
		result1 error
	}{result1}
}

func (fake *NetNS) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.doMutex.RLock()
	defer fake.doMutex.RUnlock()
	fake.setMutex.RLock()
	defer fake.setMutex.RUnlock()
	fake.pathMutex.RLock()
	defer fake.pathMutex.RUnlock()
	fake.fdMutex.RLock()
	defer fake.fdMutex.RUnlock()
	fake.closeMutex.RLock()
	defer fake.closeMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *NetNS) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}
