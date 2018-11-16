package main

import (
  "fmt"
  "math/rand"
  "time"
  "strconv"
  "github.com/Bischoff/paracuke"
)

var intResult int

func main() {
  rand.Seed(time.Now().UTC().UnixNano())

  //////////////// Steps implementation

  paracuke.When("I say \"(.*)\"", func(context string, args []string) bool {
    fmt.Printf("(%s)     %s\n", context, args[1] )
    return true
  })

  paracuke.When("I wait for a random time", func(context string, args []string) bool {
    duration := rand.Intn(10 + 1)
    fmt.Printf("(%s)     Waiting for %d seconds\n", context, duration)
    time.Sleep(time.Duration(duration) * time.Second )
    return true
  })

  paracuke.When("I wait for (.*) seconds", func(context string, args []string) bool {
    duration, _ := strconv.Atoi(args[1])
    fmt.Printf("(%s)     Waiting for %d seconds\n", context, duration)
    time.Sleep(time.Duration(duration) * time.Second )
    return true
  })

  paracuke.When("I add (.*) and (.*)", func(context string, args []string) bool {
    operand1, _ := strconv.Atoi(args[1])
    operand2, _ := strconv.Atoi(args[2])
    intResult = operand1 + operand2
    return true
  })

  paracuke.Then("I should get (.*)", func(context string, args []string) bool {
    tested, _ := strconv.Atoi(args[1])
    return intResult == tested
  })

  paracuke.ParallelTests()
}
