package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"syscall/js"
	"time"
)

const RickRollEmbed = `<iframe width="560" height="315" src="https://www.youtube.com/embed/dQw4w9WgXcQ?autoplay=1" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe>`

type player struct {
	name     string
	database string
	distance int
	avatar   *playerAvatar
}

type playerAvatar struct {
	x      int
	y      int
	width  int
	height int
	color  string
}

type race struct {
	max                int
	distance_per_pixel int
	steps              []step
}

type step struct {
	created_at time.Time
	database   string
	distance   int
}

func FetchRaceData() race {

	race := race{}

	fmt.Printf("Fetching data...\n")
	res, err := http.DefaultClient.Get("/racedata.csv")
	if err != nil {
		fmt.Printf("Error Fetching Racedata :( %+v\n", err)
	}
	defer res.Body.Close()
	csvrd := csv.NewReader(res.Body)

	summary := make(map[string]int, 0)

	for {
		cols, err := csvrd.Read()

		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Error with CSV: %+v", err)
			break
		}

		created_at, _ := time.Parse("2006-01-02 15:04:05", cols[0])
		database := cols[1]
		distance, _ := strconv.Atoi(cols[2])
		race.steps = append(race.steps, step{created_at, database, distance})
		summary[database] += distance
	}

	for _, v := range summary {
		if v > race.max {
			race.max = v
		}
	}
	race.distance_per_pixel = race.max / 600

	fmt.Printf("Content-Length: %d\n", res.ContentLength)
	fmt.Printf("Records: %d\n", len(race.steps))
	fmt.Printf("Max Value: %d\n", race.max)
	fmt.Printf("Max Value/600: %d\n", race.distance_per_pixel)
	fmt.Printf("Dump: %+v\n", summary)
	return race
}

func LoadPlayers() []*player {
	players := make([]*player, 0)
	players = append(players, &player{
		name:     js.Global().Get(fmt.Sprintf("p%dname", 1)).Get("value").String(),
		database: js.Global().Get(fmt.Sprintf("p%ddatabase", 1)).Get("value").String(),
		avatar:   &playerAvatar{x: 0, y: 10, width: 10, height: 10, color: "red"},
	})

	players = append(players, &player{
		name:     js.Global().Get(fmt.Sprintf("p%dname", 2)).Get("value").String(),
		database: js.Global().Get(fmt.Sprintf("p%ddatabase", 2)).Get("value").String(),
		avatar:   &playerAvatar{x: 0, y: 30, width: 10, height: 10, color: "blue"},
	})

	players = append(players, &player{
		name:     js.Global().Get(fmt.Sprintf("p%dname", 3)).Get("value").String(),
		database: js.Global().Get(fmt.Sprintf("p%ddatabase", 3)).Get("value").String(),
		avatar:   &playerAvatar{x: 0, y: 50, width: 10, height: 10, color: "green"},
	})
	return players
}

func main() {
	var players []*player
	race := FetchRaceData()
	runRaceC := make(chan bool)

	raceButton := js.Global().Get("raceButton")
	raceButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		players = LoadPlayers()
		for _, p := range players {
			fmt.Printf("Player: %+v, Avatar: %+v\n", p, p.avatar)
		}
		runRaceC <- true
		return nil
	}))

	for {
		<-runRaceC
		go runRace(race, players)
	}
}

func runRace(race race, players []*player) {
	canvas := js.Global().Get("track")
	canctx := canvas.Call("getContext", "2d")

	canctx.Set("fillStyle", "red")
	canctx.Call("fillRect", 0, 10, 10, 10)

	canctx.Set("fillStyle", "blue")
	canctx.Call("fillRect", 0, 30, 10, 10)

	canctx.Set("fillStyle", "green")
	canctx.Call("fillRect", 0, 50, 10, 10)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	stepIdx := 0
	stepId := js.Global().Call("setInterval", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		for j := 0; j <= 500; j++ {
			if stepIdx >= len(race.steps)-1 {
				fmt.Printf("Stepper Complete after %d out of %d!\n", stepIdx, len(race.steps)-1)
				defer wg.Done()
				return nil
			}

			stepIdx += 1
			step := race.steps[stepIdx]
			for _, p := range players {
				if step.database == p.database {
					a := p.avatar
					if a.x <= 590 {
						p.distance += step.distance

						// clear last position
						canctx.Set("fillStyle", "white")
						canctx.Call("fillRect", a.x, a.y, a.width, a.height)

						// draw new position
						a.x = p.distance / race.distance_per_pixel
						canctx.Set("fillStyle", a.color)
						canctx.Call("fillRect", a.x, a.y, a.width, a.height)
					}
				}
			}
		}
		return nil
	}), js.ValueOf(10))

	var drawFn js.Func
	drawFn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		for _, p := range players {
			a := p.avatar

			// clear last position
			canctx.Set("fillStyle", "white")
			canctx.Call("fillRect", a.x-a.width, a.y, a.width, a.height)

			// draw new position
			canctx.Set("fillStyle", a.color)
			canctx.Call("fillRect", a.x, a.y, a.width, a.height)
		}
		js.Global().Call("requestAnimationFrame", drawFn)
		return nil
	})

	fmt.Printf("stepId: %s\n", stepId)
	wg.Wait()
	fmt.Printf("Clearing Intervals\n")
	js.Global().Call("clearInterval", stepId)

	var winningPlayer player
	for _, p := range players {
		if winningPlayer.distance < p.distance {
			winningPlayer = *p
		}
		fmt.Printf("Dump: %+v\n", p)
		fmt.Printf("Dump-Avatar: %+v\n", *p.avatar)
	}
	fmt.Printf("Race Complete\n")

	DisplayWinnerDialog(winningPlayer)
}

func DisplayWinnerDialog(p player) {
	js.Global().Get("winnerMessage").Set("innerHTML", fmt.Sprintf("<h2>%s is the winner with %s!<h2>", p.name, p.database))
	js.Global().Get("rickroll").Set("innerHTML", RickRollEmbed)
	fmt.Println("0")
	dialogStyle := js.Global().Get("winnerDialog").Get("style")
	canvasStyle := js.Global().Get("track").Get("style")
	fmt.Printf("top=[%+v], left=[%+v]\n", canvasStyle.Get("top"), canvasStyle.Get("left"))
	dialogStyle.Set("top", canvasStyle.Get("top"))
	dialogStyle.Set("left", canvasStyle.Get("left"))
	dialogStyle.Set("display", canvasStyle.Get("display"))
}
