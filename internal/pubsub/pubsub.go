package pubsub

import (
    "encoding/json"
    "encoding/gob"
    "context"
    "fmt"
    "bytes"
    amqp "github.com/rabbitmq/amqp091-go"
)

// Queue types
const (
    DurableQueue = iota
    TransientQueue = iota
)

func DeclareAndBind(
    connection *amqp.Connection,
    exchange,
    queueName,
    key string,
    queueType AckType,
) (*amqp.Channel, amqp.Queue, error) {
    var queue amqp.Queue

    isDurable := queueType == DurableQueue

    connectionChannel, err := connection.Channel()
    if err != nil {
        return nil, queue, err
    }
    table := amqp.Table { "x-dead-letter-exchange": "peril_dlx" }
    queue, err = connectionChannel.QueueDeclare(queueName, isDurable, !isDurable, !isDurable, false, table)
    if err != nil {
        return nil, queue, err
    }
    if err := connectionChannel.QueueBind(queueName, key, exchange, false, nil); err != nil {
        return nil, queue, err
    }

    return connectionChannel, queue, nil
}

func handleDeliveryMessages[T any](
    deliveryChannel <-chan amqp.Delivery,
    handler func(T) AckType,
    decoder func([]byte, *T) error,
) {
    for message := range deliveryChannel {
        var body T
        if err := decoder(message.Body, &body); err != nil {
            fmt.Println("Failed to unmarshal message body")
            message.Nack(false, false) // discard it
            return
        }

        switch handler(body) {
        case AckTypeAck:
            message.Ack(false)
            fmt.Println("Message ack")
        case AckTypeNackRequeue:
            message.Nack(false, true)
            fmt.Println("Message nack requeue")
        case AckTypeNackDiscard:
            message.Nack(false, false)
            fmt.Println("Message nack discard")
        }
    }
}

type AckType = int
const (
    AckTypeAck AckType = iota
    AckTypeNackRequeue AckType = iota
    AckTypeNackDiscard AckType = iota
)

func subscribe[T any](
    connection *amqp.Connection,
    exchange,
    queueName,
    key string,
    queueType AckType,
    handler func(T) AckType,
    decoder func([]byte, *T) error,
) error {
    connectionChannel, _, err := DeclareAndBind(connection, exchange, queueName, key, queueType)
    if err != nil { return err }
    if err := connectionChannel.Qos(10, 0, false); err != nil { return err }
    deliveryChannel, err := connectionChannel.Consume(queueName, "", false, false, false, false, nil)
    if err != nil {
        connectionChannel.Close()
        return err
    }
    go handleDeliveryMessages(deliveryChannel, handler, decoder)
    return nil
}

func publish[T any](
    publishChannel *amqp.Channel,
    exchange string,
    key string,
    val T,
    contentType string,
    encoder func(T) ([]byte, error),
) error {
    bytes, err := encoder(val)
    if err != nil { return err }

    ctx := context.Background()
    publishSettings := amqp.Publishing {
        ContentType: contentType,
        Body: bytes,
    }
    if err := publishChannel.PublishWithContext(ctx, exchange, key, false, false, publishSettings); err != nil {
        return err
    }

    return nil
}

func SubscribeJSON[T any](
    connection *amqp.Connection,
    exchange,
    queueName,
    key string,
    queueType AckType,
    handler func(T) AckType,
) error {
    return subscribe(
        connection,
        exchange,
        queueName,
        key,
        queueType,
        handler,
        func (data []byte, out *T) error {
            return json.Unmarshal(data, out)
        },
    )
}

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
    return publish(
        ch,
        exchange,
        key,
        val,
        "application/json",
        func(val T) ([]byte, error) { return json.Marshal(val) },
    )
}

func SubscribeGob[T any](
    connection *amqp.Connection,
    exchange,
    queueName,
    key string,
    queueType AckType,
    handler func(T) AckType,
) error {
    return subscribe(
        connection,
        exchange,
        queueName,
        key,
        queueType,
        handler,
        func (data []byte, out *T) error {
            buffer := bytes.NewBuffer(data)
            decoder := gob.NewDecoder(buffer)
            return decoder.Decode(out)
        },
    )
}

func PublishGob[T any](ch *amqp.Channel, exchange, key string, val T) error {
    return publish(
        ch,
        exchange,
        key,
        val,
        "application/gob",
        func(val T) ([]byte, error) {
            var buffer bytes.Buffer
            encoder := gob.NewEncoder(&buffer)
            if err := encoder.Encode(&val); err != nil { return nil, err }
            return buffer.Bytes(), nil
        },
    )
}
