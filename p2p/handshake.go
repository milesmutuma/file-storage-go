package p2p

type HandshakerFunc func(any) error

func NOPHandshakeFunc(any) error { return nil }
