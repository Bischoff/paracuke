# paracuke 0.1
paracuke is a parallel cucumber written in go

  https://docs.cucumber.io

so that high testing performance is guaranteed even
with slow tests that imply waiting a lot.

It might also be used to stress a tested program
with many parallel requests.

The parallelism resides at the level of the cucumber
engine, freeing the tester from the burden of
implementing it inside the step definitions.


## Setup

To get this package, do:
```
  $ go get github.com/Bischoff/paracuke
```

To use this package:
```
  $ go get -d github.com/Bischoff/paracuke
```

To build the tests:
```
  $ cd ~/go/src/github.com/Bischoff/paracuke/
  $ go build test/stepdefinitions.go
```

To run the tests:
```
  $ ./stepdefinitions test/dummy.contexts
```

Use `-h` option to see the available command line
options.


## Usage

Test features are expressed normally:
```cucumber
Feature: Arithmetic computations

  Scenario: Basic arithmetic
    When I add 2 and 3
    Then I should get 5

  Scenario: Usual constants
    When I retrieve pi
    Then I should get 3.1415926535
```

The difference with a "normal" cucumber environment is
that the tests are executed in parallel:
```
(linux)  Feature: Arithmetic computations
(linux)  --------------------------------
(linux)
(linux)  Scenario: Basic arithmetic
(macosx)  Feature: Arithmetic computations
(macosx)  --------------------------------
(macosx)
(macosx)  Scenario: Basic arithmetic
(linux)    When I add 2 and 3
(macosx)    When I add 2 and 3
(linux)    Then I should get 5
(macosx)    Then I should get 5
```

Which test features should be run in parallel is defined in the
"contexts file":
```
linux: arithmetic.feature, system.feature
macosx: arithmetic.feature
```

If there is some preparation to be done before the parallel tests
are run, one can use the special context named `init`:
```
init: initialization.feature

linux: arithmetic.feature
macosx: arithmetic.feature, draw.feature
```

Similarly, termination work can be done in the special context
named `end`.


## Steps implementation

Steps are implemented in go, following the model:
```go
paracuke.When("I add (.*) and (.*)", func(context *paracuke.Context, args []string) bool {
  operand1, _ := strconv.Atoi(args[1])
  operand2, _ := strconv.Atoi(args[2])
  context.Data["result"] = strconv.Itoa(operand1 + operand2)
  return true
})
```

The variable parts of the regular expression are stored in the `args`
array of character strings. For example, with `When I add 2 and 3`,
`args[1]` contains `"2"`, and `args[2]` contains `"3"`.

The step definition function returns a boolean. `false` means that the
test has failed.

Data can be shared between the various step implementations inside
the same context, by using the `context.Data` map of character strings.
In the example above, we use `context.Data["result"]` to store the
result of the addition, so it can be reused in the `Then I should get 5`
step. Context data are specific to a given context, and will not leak
to the other contexts.

The name of the context can be accessed through the predefined map entry
`context.Data["name"]`:
```go
paracuke.When("I say \"(.*)\"", func(context *paracuke.Context, args []string) bool {
  name := context.Data["name"]
  fmt.Printf("(%s)     %s\n", name, args[1] )
  return true
})
```

For example, with `When I say "Hurray!"`, that step definition would
print out `(macosx)     Hurray!` when run in the context named `macosx`.

At the end of your step definitions, start the paracuke engine:
```go
  paracuke.ParallelTests()
```


## Comparaison with other parallel cucumber solutions

In paracuke, the atomic unit is the test step (`Given`, `When`,
or `Then`). This means that the context might change between two steps
of the same scenario, unlike parallel-cucumber-js:

  https://github.com/simondean/parallel-cucumber-js

where the atomic unit is the scenario.

The parallelism in paracuke is acheived through coroutines, whereas
parallel-cucumber-js uses preemptive multitasking workers.

Manual solutions include using several parallel Java Virtual Machines
as shown in the article:

  https://automationrhapsody.com/running-cucumber-tests-in-parallel/

either in separate processes or threads.


## Possible future developments

### Core extensions

Instead of a boolean, the steps could return a value out of three,
meaning: success, failure, or skipped.

There should be a command-line option to ignore one or more contexts:
```
  -i init,end
```
would ignore the special initialization and termination contexts
`init` and `end`.

The syntax of the contexts file could be changed to yaml or XML.

There is no final results report printed yet.

JUnit reports currently cannot be generated.

Performance could be increased by reading all features before starting
to execute them.


### Cucumber facilities

Conditional tags are not supported yet:
```cucumber
@powerpc
  Scenario: Basic arithmetic
```

Similarly, there is no feature nor scenario hooks yet.

There is no `Background` section,
nor `Scenario outline` and `Examples`.


### Property-based testing

paracuke could be extended to generate tests automatically
and reduce them to a minimal failing sequence:

  https://www.youtube.com/watch?v=hXnS_Xjwk2Y

