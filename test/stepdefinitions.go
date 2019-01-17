package main

import (
  "fmt"
  "math/rand"
  "time"
  "strconv"
  "github.com/Bischoff/paracuke"
)

func main() {
  rand.Seed(time.Now().UTC().UnixNano())

  //////////////// Steps implementation

  paracuke.When("^I say \"(.*)\"$", func(context *paracuke.Context, args []string) bool {
    name := context.Data["name"]
    fmt.Printf("(%s)     %s\n", name, args[1] )
    return true
  })

  paracuke.When("^I wait for a random time$", func(context *paracuke.Context, args []string) bool {
    name := context.Data["name"]
    duration := rand.Intn(10 + 1)
    fmt.Printf("(%s)     Waiting for %d seconds\n", name, duration)
    time.Sleep(time.Duration(duration) * time.Second )
    return true
  })

  paracuke.When("^I wait for (.*) seconds$", func(context *paracuke.Context, args []string) bool {
    name := context.Data["name"]
    duration, _ := strconv.Atoi(args[1])
    fmt.Printf("(%s)     Waiting for %d seconds\n", name, duration)
    time.Sleep(time.Duration(duration) * time.Second )
    return true
  })

  paracuke.When("^I add (.*) and (.*)$", func(context *paracuke.Context, args []string) bool {
    operand1, _ := strconv.Atoi(args[1])
    operand2, _ := strconv.Atoi(args[2])
    context.Data["result"] = strconv.Itoa(operand1 + operand2)
    return true
  })

  paracuke.Then("^I should get (.*)$", func(context *paracuke.Context, args []string) bool {
    tested, _ := strconv.Atoi(args[1])
    result, _ := strconv.Atoi(context.Data["result"])
    return result == tested
  })

  paracuke.ParallelTests()
}
