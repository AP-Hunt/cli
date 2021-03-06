// Code generated by counterfeiter. DO NOT EDIT.
package v3actionfakes

import (
	"sync"

	"code.cloudfoundry.org/cli/actor/v3action"
	"code.cloudfoundry.org/cli/api/uaa/constant"
)

type FakeUAAClient struct {
	AuthenticateStub        func(map[string]string, string, constant.GrantType) (string, string, error)
	authenticateMutex       sync.RWMutex
	authenticateArgsForCall []struct {
		arg1 map[string]string
		arg2 string
		arg3 constant.GrantType
	}
	authenticateReturns struct {
		result1 string
		result2 string
		result3 error
	}
	authenticateReturnsOnCall map[int]struct {
		result1 string
		result2 string
		result3 error
	}
	GetSSHPasscodeStub        func(string, string) (string, error)
	getSSHPasscodeMutex       sync.RWMutex
	getSSHPasscodeArgsForCall []struct {
		arg1 string
		arg2 string
	}
	getSSHPasscodeReturns struct {
		result1 string
		result2 error
	}
	getSSHPasscodeReturnsOnCall map[int]struct {
		result1 string
		result2 error
	}
	LoginPromptsStub        func() map[string][]string
	loginPromptsMutex       sync.RWMutex
	loginPromptsArgsForCall []struct {
	}
	loginPromptsReturns struct {
		result1 map[string][]string
	}
	loginPromptsReturnsOnCall map[int]struct {
		result1 map[string][]string
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeUAAClient) Authenticate(arg1 map[string]string, arg2 string, arg3 constant.GrantType) (string, string, error) {
	fake.authenticateMutex.Lock()
	ret, specificReturn := fake.authenticateReturnsOnCall[len(fake.authenticateArgsForCall)]
	fake.authenticateArgsForCall = append(fake.authenticateArgsForCall, struct {
		arg1 map[string]string
		arg2 string
		arg3 constant.GrantType
	}{arg1, arg2, arg3})
	fake.recordInvocation("Authenticate", []interface{}{arg1, arg2, arg3})
	fake.authenticateMutex.Unlock()
	if fake.AuthenticateStub != nil {
		return fake.AuthenticateStub(arg1, arg2, arg3)
	}
	if specificReturn {
		return ret.result1, ret.result2, ret.result3
	}
	fakeReturns := fake.authenticateReturns
	return fakeReturns.result1, fakeReturns.result2, fakeReturns.result3
}

func (fake *FakeUAAClient) AuthenticateCallCount() int {
	fake.authenticateMutex.RLock()
	defer fake.authenticateMutex.RUnlock()
	return len(fake.authenticateArgsForCall)
}

func (fake *FakeUAAClient) AuthenticateCalls(stub func(map[string]string, string, constant.GrantType) (string, string, error)) {
	fake.authenticateMutex.Lock()
	defer fake.authenticateMutex.Unlock()
	fake.AuthenticateStub = stub
}

func (fake *FakeUAAClient) AuthenticateArgsForCall(i int) (map[string]string, string, constant.GrantType) {
	fake.authenticateMutex.RLock()
	defer fake.authenticateMutex.RUnlock()
	argsForCall := fake.authenticateArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2, argsForCall.arg3
}

func (fake *FakeUAAClient) AuthenticateReturns(result1 string, result2 string, result3 error) {
	fake.authenticateMutex.Lock()
	defer fake.authenticateMutex.Unlock()
	fake.AuthenticateStub = nil
	fake.authenticateReturns = struct {
		result1 string
		result2 string
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeUAAClient) AuthenticateReturnsOnCall(i int, result1 string, result2 string, result3 error) {
	fake.authenticateMutex.Lock()
	defer fake.authenticateMutex.Unlock()
	fake.AuthenticateStub = nil
	if fake.authenticateReturnsOnCall == nil {
		fake.authenticateReturnsOnCall = make(map[int]struct {
			result1 string
			result2 string
			result3 error
		})
	}
	fake.authenticateReturnsOnCall[i] = struct {
		result1 string
		result2 string
		result3 error
	}{result1, result2, result3}
}

func (fake *FakeUAAClient) GetSSHPasscode(arg1 string, arg2 string) (string, error) {
	fake.getSSHPasscodeMutex.Lock()
	ret, specificReturn := fake.getSSHPasscodeReturnsOnCall[len(fake.getSSHPasscodeArgsForCall)]
	fake.getSSHPasscodeArgsForCall = append(fake.getSSHPasscodeArgsForCall, struct {
		arg1 string
		arg2 string
	}{arg1, arg2})
	fake.recordInvocation("GetSSHPasscode", []interface{}{arg1, arg2})
	fake.getSSHPasscodeMutex.Unlock()
	if fake.GetSSHPasscodeStub != nil {
		return fake.GetSSHPasscodeStub(arg1, arg2)
	}
	if specificReturn {
		return ret.result1, ret.result2
	}
	fakeReturns := fake.getSSHPasscodeReturns
	return fakeReturns.result1, fakeReturns.result2
}

func (fake *FakeUAAClient) GetSSHPasscodeCallCount() int {
	fake.getSSHPasscodeMutex.RLock()
	defer fake.getSSHPasscodeMutex.RUnlock()
	return len(fake.getSSHPasscodeArgsForCall)
}

func (fake *FakeUAAClient) GetSSHPasscodeCalls(stub func(string, string) (string, error)) {
	fake.getSSHPasscodeMutex.Lock()
	defer fake.getSSHPasscodeMutex.Unlock()
	fake.GetSSHPasscodeStub = stub
}

func (fake *FakeUAAClient) GetSSHPasscodeArgsForCall(i int) (string, string) {
	fake.getSSHPasscodeMutex.RLock()
	defer fake.getSSHPasscodeMutex.RUnlock()
	argsForCall := fake.getSSHPasscodeArgsForCall[i]
	return argsForCall.arg1, argsForCall.arg2
}

func (fake *FakeUAAClient) GetSSHPasscodeReturns(result1 string, result2 error) {
	fake.getSSHPasscodeMutex.Lock()
	defer fake.getSSHPasscodeMutex.Unlock()
	fake.GetSSHPasscodeStub = nil
	fake.getSSHPasscodeReturns = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeUAAClient) GetSSHPasscodeReturnsOnCall(i int, result1 string, result2 error) {
	fake.getSSHPasscodeMutex.Lock()
	defer fake.getSSHPasscodeMutex.Unlock()
	fake.GetSSHPasscodeStub = nil
	if fake.getSSHPasscodeReturnsOnCall == nil {
		fake.getSSHPasscodeReturnsOnCall = make(map[int]struct {
			result1 string
			result2 error
		})
	}
	fake.getSSHPasscodeReturnsOnCall[i] = struct {
		result1 string
		result2 error
	}{result1, result2}
}

func (fake *FakeUAAClient) LoginPrompts() map[string][]string {
	fake.loginPromptsMutex.Lock()
	ret, specificReturn := fake.loginPromptsReturnsOnCall[len(fake.loginPromptsArgsForCall)]
	fake.loginPromptsArgsForCall = append(fake.loginPromptsArgsForCall, struct {
	}{})
	fake.recordInvocation("LoginPrompts", []interface{}{})
	fake.loginPromptsMutex.Unlock()
	if fake.LoginPromptsStub != nil {
		return fake.LoginPromptsStub()
	}
	if specificReturn {
		return ret.result1
	}
	fakeReturns := fake.loginPromptsReturns
	return fakeReturns.result1
}

func (fake *FakeUAAClient) LoginPromptsCallCount() int {
	fake.loginPromptsMutex.RLock()
	defer fake.loginPromptsMutex.RUnlock()
	return len(fake.loginPromptsArgsForCall)
}

func (fake *FakeUAAClient) LoginPromptsCalls(stub func() map[string][]string) {
	fake.loginPromptsMutex.Lock()
	defer fake.loginPromptsMutex.Unlock()
	fake.LoginPromptsStub = stub
}

func (fake *FakeUAAClient) LoginPromptsReturns(result1 map[string][]string) {
	fake.loginPromptsMutex.Lock()
	defer fake.loginPromptsMutex.Unlock()
	fake.LoginPromptsStub = nil
	fake.loginPromptsReturns = struct {
		result1 map[string][]string
	}{result1}
}

func (fake *FakeUAAClient) LoginPromptsReturnsOnCall(i int, result1 map[string][]string) {
	fake.loginPromptsMutex.Lock()
	defer fake.loginPromptsMutex.Unlock()
	fake.LoginPromptsStub = nil
	if fake.loginPromptsReturnsOnCall == nil {
		fake.loginPromptsReturnsOnCall = make(map[int]struct {
			result1 map[string][]string
		})
	}
	fake.loginPromptsReturnsOnCall[i] = struct {
		result1 map[string][]string
	}{result1}
}

func (fake *FakeUAAClient) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.authenticateMutex.RLock()
	defer fake.authenticateMutex.RUnlock()
	fake.getSSHPasscodeMutex.RLock()
	defer fake.getSSHPasscodeMutex.RUnlock()
	fake.loginPromptsMutex.RLock()
	defer fake.loginPromptsMutex.RUnlock()
	copiedInvocations := map[string][][]interface{}{}
	for key, value := range fake.invocations {
		copiedInvocations[key] = value
	}
	return copiedInvocations
}

func (fake *FakeUAAClient) recordInvocation(key string, args []interface{}) {
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

var _ v3action.UAAClient = new(FakeUAAClient)
