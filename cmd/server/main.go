package main

import (
    "fmt"
    "github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
    "github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
    "github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
    amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
    gamelogic.PrintServerHelp()

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

    repl:
    for {
        input := gamelogic.GetInput()
        if len(input) == 0 { continue }

        switch input[0] {

        case "pause":
            fmt.Println("Sending pause message")
            err = pubsub.PublishJSON(
                connectionChannel,
                routing.ExchangePerilDirect,
                routing.PauseKey,
                routing.PlayingState { IsPaused: true },
            )
            if err != nil {
                fmt.Println("Failed to publish pause message to exchange")
            }

        case "resume":
            fmt.Println("Sending resume message")
            err = pubsub.PublishJSON(
                connectionChannel,
                routing.ExchangePerilDirect,
                routing.PauseKey,
                routing.PlayingState { IsPaused: false },
            )
            if err != nil {
                fmt.Println("Failed to publish resume message to exchange")
            }

        case "quit":
            fmt.Println("Exiting")
            break repl

        default:
            fmt.Printf("Unrecognized command: %v\n", input[0])
        }
    }
}
