package main

import (
    "fmt"
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

    gamestate := gamelogic.NewGameState(username)

    repl:
    for {
        input := gamelogic.GetInput()
        if len(input) == 0 { continue }

        switch input[0] {

        case "spawn":
            if err := gamestate.CommandSpawn(input); err != nil {
                fmt.Printf("Failed to spawn: %v\n", err)
            }

        case "move":
            _, err := gamestate.CommandMove(input)
            if err != nil {
                fmt.Printf("Failed to move: %v\n", err)
            } else {
                fmt.Println("Move successful")
            }

        case "status":
            gamestate.CommandStatus()

        case "help":
            gamelogic.PrintClientHelp()

        case "spam":
            fmt.Println("Spamming not allowed yet!")

        case "quit":
            gamelogic.PrintQuit()
            break repl

        default:
            fmt.Println("Unrecognized command")

        }
    }
}
