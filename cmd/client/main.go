package main

import (
    "fmt"
    "github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
    "github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
    "github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
    amqp "github.com/rabbitmq/amqp091-go"
)

type PauseHandler = func(routing.PlayingState) pubsub.AckType
func handlerPause(gs *gamelogic.GameState) PauseHandler {
    return func(ps routing.PlayingState) pubsub.AckType {
        defer fmt.Print("> ")
        gs.HandlePause(ps)
        return pubsub.AckTypeAck
    }
}

type MoveHandler = func(gamelogic.ArmyMove) pubsub.AckType
func handlerMove(gs *gamelogic.GameState, publishChannel *amqp.Channel) MoveHandler {
    return func(move gamelogic.ArmyMove) pubsub.AckType {
        defer fmt.Print("> ")
        switch gs.HandleMove(move) {
        case gamelogic.MoveOutComeSafe: return pubsub.AckTypeAck

        case gamelogic.MoveOutcomeMakeWar:
            err := pubsub.PublishJSON(
                publishChannel,
                routing.ExchangePerilTopic,
                fmt.Sprintf("%v.%v", routing.WarRecognitionsPrefix, gs.GetUsername()),
                gamelogic.RecognitionOfWar {
                    Attacker: move.Player,
                    Defender: gs.GetPlayerSnap(),
                },
            )
            if err != nil {
                fmt.Printf("failed to publish war recognition: %v\n", err)
                return pubsub.AckTypeNackRequeue
            }
            return pubsub.AckTypeAck

        default: return pubsub.AckTypeNackDiscard
        }
    }
}

type WarHandler = func(gamelogic.RecognitionOfWar) pubsub.AckType
func handlerWar(gs *gamelogic.GameState) WarHandler {
    return func(warDecl gamelogic.RecognitionOfWar) pubsub.AckType {
        defer fmt.Printf("> ")
        outcome, _, _ := gs.HandleWar(warDecl)
        switch outcome {
        case gamelogic.WarOutcomeNotInvolved: return pubsub.AckTypeNackRequeue
        case gamelogic.WarOutcomeNoUnits: return pubsub.AckTypeNackDiscard
        case gamelogic.WarOutcomeOpponentWon: fallthrough
        case gamelogic.WarOutcomeYouWon: fallthrough
        case gamelogic.WarOutcomeDraw: return pubsub.AckTypeAck
        default:
            fmt.Println("Unrecognized war outcome")
            return pubsub.AckTypeNackDiscard
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

    publishChannel, err := connection.Channel()
    if err != nil {
        fmt.Printf("failed to get publish channel: %v\n", err)
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
        handlerMove(gamestate, publishChannel),
    )
    pubsub.SubscribeJSON(
        connection,
        routing.ExchangePerilTopic,
        routing.WarRecognitionsPrefix,
        fmt.Sprintf("%v.*", routing.WarRecognitionsPrefix),
        pubsub.DurableQueue,
        handlerWar(gamestate),
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
                    publishChannel,
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
