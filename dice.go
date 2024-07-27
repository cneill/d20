package main

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"slices"
	"sort"
	"strings"
	"time"
)

type Die struct {
	Sides int
}

func (d Die) Roll() int {
	// Don't want this to return 0 so we cap at sides-1 and add 1
	val, err := rand.Int(rand.Reader, big.NewInt(int64(d.Sides)))
	if err != nil {
		panic(fmt.Errorf("failed to roll die: %w", err))
	}

	return int(val.Int64()) + 1
}

type Dice []Die

func NewDice(sides, num int) Dice {
	result := make(Dice, num)

	for i := range num {
		result[i] = Die{sides}
	}

	return result
}

func (d Dice) Roll(user *User) Roll {
	result := make([]int, len(d))

	for i, die := range d {
		result[i] = die.Roll()
	}

	roll := Roll{
		Result: result,
		Time:   time.Now(),
		User:   user,
	}

	return roll
}

type Roll struct {
	Result DiceResults
	Time   time.Time
	User   *User
}

type Rolls []Roll

func (r Rolls) Sort() Rolls {
	newRolls := slices.Clone(r)
	sort.Slice(newRolls, func(i, j int) bool {
		return newRolls[i].Time.After(newRolls[j].Time)
	})

	return newRolls
}

type DiceResults []int

func (r DiceResults) String() string {
	result := ""

	for _, dieResult := range r {
		result += fmt.Sprintf("%d | ", dieResult)
	}

	result = strings.TrimSuffix(result, " | ")

	return result
}
