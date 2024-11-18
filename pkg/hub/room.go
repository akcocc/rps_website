package hub

import (
	"context"
	"fmt"
	"rps_website/pkg/assert"
	"strconv"
)

type Room struct {
    players [2]*Client
}

type Buf []byte
func (buf *Buf) Write(p []byte) (n int, err error) {
    *buf = p
    return len(p), nil
}

func (room *Room) mediate() {
    println("mediating")
    player_1 := room.players[0]
    player_1.in_match = true
    player_2 := room.players[1]
    player_2.in_match = true

    assert.Assert_ne(
        player_1.connection.RemoteAddr(),
        player_2.connection.RemoteAddr(),
        "ADDRESSES SHOULD NOT BE EQUAL",
        )


    var game_screen1 Buf
    play_screen(player_2.player_name).Render(context.Background(), &game_screen1)
    player_1.message_channel <- game_screen1

    // I'm using this because there's some kind of race condition with the
    // server writing to the clients at the same time, causing them to have the
    // same game screen where both players will see "Playing against {name}"
    // where `name` is the same for both players, which shouldn't be the case.
    // go run -race doesn't pick up this race condition and I'm not sure how
    // else to solve it...
    <-player_1.message_channel

    var game_screen2 Buf
    play_screen(player_1.player_name).Render(context.Background(), &game_screen2)
    player_2.message_channel <- game_screen2

    <-player_2.message_channel

    p1_action, p2_action := 0, 0

    for p1_action == 0 || p2_action == 0 {
        select {
        case client_message := <-player_1.message_channel:
            if string(client_message) == "left" {
                println("player_1 leaving")
                player_2.send_player_left_screen(player_1.player_name)
                room.players[0] = nil
                player_2.in_match = false
                break
            }
            p1_action, err := strconv.Atoi(string(client_message))
            assert.Expect(err, "could not parse message data as json")
            fmt.Printf("action: %d\n", p1_action)

        case _ = <-player_2.message_channel:
            println("player_2 leaving")
            player_1.send_player_left_screen(player_2.player_name)
            player_1.in_match = false
            room.players[1] = nil
        }
    }
}
