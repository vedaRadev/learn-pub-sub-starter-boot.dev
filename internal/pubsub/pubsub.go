package pubsub

import (
    "encoding/json"
    "context"
    "fmt"
    amqp "github.com/rabbitmq/amqp091-go"
)

func PublishJSON[T any](ch *amqp.Channel, exchange, key string, val T) error {
    jsonBytes, err := json.Marshal(val)
    if err != nil { return err }

    ctx := context.Background()
    publishSettings := amqp.Publishing {
        ContentType: "application/json",
        Body: jsonBytes,
    }
    if err := ch.PublishWithContext(ctx, exchange, key, false, false, publishSettings); err != nil {
        return err
    }

    return nil
}

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
    queueType int,
) (*amqp.Channel, amqp.Queue, error) {
    var queue amqp.Queue

    isDurable := queueType == DurableQueue

    connectionChannel, err := connection.Channel()
    if err != nil {
        return nil, queue, err
    }
    queue, err = connectionChannel.QueueDeclare(queueName, isDurable, !isDurable, !isDurable, false, nil)
    if err != nil {
        return nil, queue, err
    }
    if err := connectionChannel.QueueBind(queueName, key, exchange, false, nil); err != nil {
        return nil, queue, err
    }

    return connectionChannel, queue, nil
}

func handleDeliveryMessages[T any](deliveryChannel <-chan amqp.Delivery, handler func(T)) {
    for message := range deliveryChannel {
        var body T
        if err := json.Unmarshal(message.Body, &body); err != nil {
            fmt.Println("Failed to unmarshal message body")
        } else {
            handler(body)
        }
        message.Ack(false)
    }
}

func SubscribeJSON[T any](
    connection *amqp.Connection,
    exchange,
    queueName,
    key string,
    queueType int,
    handler func(T),
) error {
    connectionChannel, _, err := DeclareAndBind(connection, exchange, queueName, key, queueType)
    if err != nil { return err }

    deliveryChannel, err := connectionChannel.Consume(queueName, "", false, false, false, false, nil)
    if err != nil {
        connectionChannel.Close()
        return err
    }

    go handleDeliveryMessages(deliveryChannel, handler)

    return nil
}
