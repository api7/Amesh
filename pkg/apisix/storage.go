package apisix

type Storage interface {
	Store(string, string)
}
