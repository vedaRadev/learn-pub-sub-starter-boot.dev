package pubsub

import (
    "encoding/json"
    "context"
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

const (
    DurableQueue = iota
    TransientQueue = iota
)

func DeclareAndBind(
    connection *amqp.Connection,
    exchange,
    queueName,
    key string,
    simpleQueueType int, // enum to repr "durable" or "transient"
) (*amqp.Channel, amqp.Queue, error) {
    var queue amqp.Queue

    isDurable := simpleQueueType == DurableQueue

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
