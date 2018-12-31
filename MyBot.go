package main

import (
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"./hlt"
	"./hlt/gameconfig"
	"./hlt/log"
)

func gracefulExit(logger *log.FileLogger) {
	var gracefulStop = make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)
	go func() {
		sig := <-gracefulStop
		fmt.Printf("caught sig: %+v", sig)
		fmt.Println("Wait for 2 second to finish processing")
		time.Sleep(2 * time.Second)
		logger.Close()
		os.Exit(0)
	}()
}

func main() {
	args := os.Args
	var seed = time.Now().UnixNano() % int64(os.Getpid())
	if len(args) > 1 {
		seed, _ = strconv.ParseInt(args[0], 10, 64)
	}
	rand.Seed(seed)

	var game = hlt.NewGame()
	// At this point "game" variable is populated with initial map data.
	// This is a good place to do computationally expensive start-up pre-processing.
	// As soon as you call "ready" function below, the 2 second per turn timer will start.

	var config = gameconfig.GetInstance()
	fileLogger := log.NewFileLogger(game.Me.ID)
	var logger = fileLogger.Logger
	logger.Printf("Successfully created bot! My Player ID is %d. Bot rng seed is %d.", game.Me.ID, seed)
	gracefulExit(fileLogger)
	game.Ready("MyBot")

	for {
		game.UpdateFrame()
		var me = game.Me
		var gameMap = game.Map
		var ships = me.Ships
		var commands = []hlt.Command{}
		for i := range ships {
			var ship = ships[i]
			var maxHalite, _ = config.GetInt(gameconfig.MaxHalite)
			var currentCell = gameMap.AtEntity(ship.E)
			if ship.IsFull() || float64(ship.Halite) >= float64(maxHalite)*(float64(3)/float64(4)) {
				commands = append(commands, ship.Move(gameMap.NaiveNavigate(ship, gameMap.Normalize(me.Shipyard.E.Pos))))
				logger.Printf("shipper's full")
			} else if currentCell.Halite < (maxHalite / 10) {
				direction := getSafeDirection(ship, gameMap)
				commands = append(commands, ship.Move(direction))
				logger.Printf("oscar mike")
			} else {
				commands = append(commands, ship.Move(hlt.Still()))
				logger.Printf("tired turtle")
			}
		}
		var shipCost, _ = config.GetInt(gameconfig.ShipCost)
		if game.TurnNumber <= 200 && me.Halite >= shipCost && !gameMap.AtEntity(me.Shipyard.E).IsOccupied() {
			commands = append(commands, hlt.SpawnShip{})
		}
		game.EndTurn(commands)
	}

}

func getSafeDirection(ship *hlt.Ship, gameMap *hlt.GameMap) *hlt.Direction {
	var safe = false
	var direction *hlt.Direction
	var offsetPosition *hlt.Position
	var newMapCell *hlt.MapCell

	for !safe {
		direction = hlt.AllDirections[rand.Intn(4)]
		offsetPosition, _ = ship.E.Pos.DirectionalOffset(direction)
		newMapCell = gameMap.AtPosition(gameMap.Normalize(offsetPosition))
		if newMapCell.IsEmpty() {
			newMapCell.MarkUnsafe(ship)
			safe = true
		}
	}
	// fmt.Printf("%+v\n", direction)
	return direction
}

// might be useful in the future - used in a test bot that wound up not being better than random moves
func getMaxHaliteDirection(ship *hlt.Ship, gameMap *hlt.GameMap, game *hlt.Game) *hlt.Direction {
	var offsetDirection *hlt.Position
	var newMapCell *hlt.MapCell
	var haliteAmt int
	var maxDirection *hlt.Direction = hlt.Still()

	for _, direction := range hlt.AllDirections {
		offsetDirection, _ = ship.E.Pos.DirectionalOffset(direction)
		newMapCell = gameMap.AtPosition(offsetDirection)

		if newMapCell.Halite > haliteAmt && isSafeDirection(ship, gameMap, direction) {
			haliteAmt = newMapCell.Halite
			maxDirection = direction
		}
	}

	return maxDirection
}

func isSafeDirection(ship *hlt.Ship, gameMap *hlt.GameMap, direction *hlt.Direction) bool {
	var offsetPosition *hlt.Position
	var newMapCell *hlt.MapCell

	offsetPosition, _ = ship.E.Pos.DirectionalOffset(direction)
	newMapCell = gameMap.AtPosition(offsetPosition)
	if !newMapCell.IsOccupied() {
		return true
	}

	return false
}
