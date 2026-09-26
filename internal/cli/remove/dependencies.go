package remove

type CheckoutDirs interface {
	Remove(string) error
}
