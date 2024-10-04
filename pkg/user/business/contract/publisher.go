package contract

type Publisher interface {
	Publish(topic string, msg interface{}) error
}
