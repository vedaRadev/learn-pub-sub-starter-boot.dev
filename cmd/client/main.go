package main

import (
    "fmt"
    "github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
    "github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
    "github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
    amqp "github.com/rabbitmq/amqp091-go"
)

func handlerPause(gs *gamelogic.GameState) func(routing.PlayingState) pubsub.AckType {
    return func(ps routing.PlayingState) pubsub.AckType {
        defer fmt.Print("> ")
        gs.HandlePause(ps)
        return pubsub.AckTypeAck
    }
}

func handlerMove(gs *gamelogic.GameState) func(gamelogic.ArmyMove) pubsub.AckType {
    return func(move gamelogic.ArmyMove) pubsub.AckType {
        defer fmt.Print("> ")
        switch gs.HandleMove(move) {
        case gamelogic.MoveOutComeSafe: fallthrough
        case gamelogic.MoveOutcomeMakeWar: return pubsub.AckTypeAck
        default: return pubsub.AckTypeNackDiscard
        }
    }
}

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

    _, _, err = pubsub.DeclareAndBind(
        connection,
        routing.ExchangePerilDirect,
        fmt.Sprintf("%v.%v", routing.PauseKey, username),
        routing.PauseKey,
        pubsub.TransientQueue,
    )
    if err != nil {
        fmt.Printf("Failed to create and bind queue to connection: %v\n", err)
        return
    }

    userMovesChannel, _, err := pubsub.DeclareAndBind(
        connection,
        routing.ExchangePerilTopic,
        fmt.Sprintf("%v.%v", routing.ArmyMovesPrefix, username),
        fmt.Sprintf("%v.*", routing.ArmyMovesPrefix),
        pubsub.TransientQueue,
    )
    if err != nil {
        fmt.Printf("Failed to create and bind moves queue: %v\n", err)
        return
    }

    gamestate := gamelogic.NewGameState(username)
    pubsub.SubscribeJSON(
        connection,
        routing.ExchangePerilDirect,
        fmt.Sprintf("pause.%v", username),
        routing.PauseKey,
        pubsub.TransientQueue,
        handlerPause(gamestate),
    )
    pubsub.SubscribeJSON(
        connection,
        routing.ExchangePerilTopic,
        fmt.Sprintf("%v.%v", routing.ArmyMovesPrefix, username),
        fmt.Sprintf("%v.*", routing.ArmyMovesPrefix),
        pubsub.TransientQueue,
        handlerMove(gamestate),
    )

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
            move, err := gamestate.CommandMove(input)
            if err != nil {
                fmt.Printf("Failed to move: %v\n", err)
            } else {
                err := pubsub.PublishJSON(
                    userMovesChannel,
                    routing.ExchangePerilTopic,
                    fmt.Sprintf("%v.%v", routing.ArmyMovesPrefix, username),
                    move,
                )
                if err != nil {
                    fmt.Printf("Failed to publish move: %v\n", err)
                } else {
                    fmt.Println("Published move")
                }
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
