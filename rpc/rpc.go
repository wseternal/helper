package rpc

type MethodHandler func(req interface{}) (res interface{}, err error)

type Service interface {
	RegisterMethod(uid string, h MethodHandler)
	SetPrivateData(data interface{})
	GetPrivateData() interface{}
}
