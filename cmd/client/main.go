package main

import (
    "fmt"
    "os"
    "os/signal"
    "github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
    "github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
    "github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
    amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
    username, err := gamelogic.ClientWelcome()
    if err != nil {
        fmt.Println("Failed to get username")
        return
    }

    connectionString := "amqp://guest:guest@localhost:5672/"
    connection, err := amqp.Dial(connectionString)
    if err != nil {
        fmt.Println("Failed to connect to rabbitmq server")
        return
    }
    defer connection.Close()
    fmt.Println("Connected to rabbitmq server")

    // connectionChannel, queue, err := pubsub.DeclareAndBind(
    _, _, err = pubsub.DeclareAndBind(
        connection,
        routing.ExchangePerilDirect,
        fmt.Sprintf("%v.%v", routing.PauseKey, username),
        routing.PauseKey,
        pubsub.TransientQueue,
    )
    if err != nil {
        fmt.Printf("Failed to create and bind queue to connection: %v", err)
        return
    }

    osSignals := make(chan os.Signal, 1)
    signal.Notify(osSignals, os.Interrupt)
    <-osSignals
    fmt.Println("Shutting down and closing rabbitmq server connection")
}
