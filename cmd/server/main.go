package main

import (
    "fmt"
    "os"
    "os/signal"
    "github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
    "github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
    amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	fmt.Println("Starting Peril server...")
    connectionString := "amqp://guest:guest@localhost:5672/"
    connection, err := amqp.Dial(connectionString)
    if err != nil {
        fmt.Println("Failed to connect to rabbitmq server")
        return
    }
    defer connection.Close()
    fmt.Println("Connected to rabbitmq server")

    connectionChannel, err := connection.Channel()
    if err != nil {
        fmt.Println("Failed to make channel from rabbitmq connection")
        return
    }

    err = pubsub.PublishJSON(
        connectionChannel,
        routing.ExchangePerilDirect,
        routing.PauseKey,
        routing.PlayingState { IsPaused: true },
    )

    osSignals := make(chan os.Signal, 1)
    signal.Notify(osSignals, os.Interrupt)
    <-osSignals
    fmt.Println("Shutting down and closing rabbitmq server connection")
}
