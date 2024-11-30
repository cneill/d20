package main

import (
	"fmt"
	"math/rand/v2"
	"sort"
	"strings"
	"time"
)

type Die struct {
	Sides          int
	CritOn         int // at or below
	ComplicationOn int // at or above
}

func (d Die) Roll() DieResult {
	value := 1 + rand.IntN(d.Sides)
	crit := false
	complication := false

	if value <= d.CritOn {
		crit = true
	}

	if value >= d.ComplicationOn {
		complication = true
	}

	return DieResult{
		Value:        value,
		Crit:         crit,
		Complication: complication,
	}
}

type Dice []Die

func NewDice(sides, num, critOn, complicationOn int) Dice {
	result := make(Dice, num)

	for i := range num {
		result[i] = Die{
			Sides:          sides,
			CritOn:         critOn,
			ComplicationOn: complicationOn,
		}
	}

	return result
}

func (d Dice) Roll(user *User) Roll {
	result := make([]DieResult, len(d))

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
	newRolls := make(Rolls, len(r))

	for rollNum := range r {
		newRoll := Roll{
			Result: r[rollNum].Result,
			Time:   r[rollNum].Time,
			User:   r[rollNum].User,
		}
		newRolls[rollNum] = newRoll
	}

	sort.Slice(newRolls, func(i, j int) bool {
		return newRolls[i].Time.After(newRolls[j].Time)
	})

	return newRolls
}

type DieResult struct {
	Value        int
	Crit         bool
	Complication bool
}

type DiceResults []DieResult

func (d DiceResults) String() string {
	result := ""

	for _, dieResult := range d {
		result += fmt.Sprintf("%d | ", dieResult.Value)
	}

	result = strings.TrimSuffix(result, " | ")

	return result
}
