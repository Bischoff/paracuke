# paracuke reference documentation

## Setup

To get paracuke, do:
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
```yaml
- batch:
  - context:
      data:
        name: linux
      features:
        - arithmetic.feature
        - system.feature
  - context:
      data:
        name: macosx
      features:
        - arithmetic.feature
```


## Contexts file syntax

The contexts file follows YAML syntax.

Batches are executed in sequence, and all parallel contexts
of a batch must terminate before the next batch is executed.
This enables for example to run some initialization before
the real tests begin:
```yaml
- batch:
  - context:
      data:
        name: init
      features:
        - initialization.feature

- batch:
  - context:
      data:
        name: linux
      features:
(etc.)
```

The contexts are run in parallel. A context is made of data and
features. One data that should always be present is the `name`
of the context. Other data can be used to store context-specific
variables:
```yaml
- batch:
  - context:
      data:
        name: linux
        founder: Linus Torvalds
      features:
        - arithmetic.feature
        - system.feature
  - context:
      data:
        name: macosx
        founder: Steve Wozniak, Steve Jobs
      features:
        - arithmetic.feature
```

In this example, both contexts will know about a variable named `founder`.

Inside of a context, the features are run in sequence. They are
given by their file name, either absolute, or relative to the
tests executable.


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

The name of the context can be accessed through `context.Data["name"]`:
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
  paracuke.RunTests()
```

Similarly, other data can be access via the `Data` map:
```go
  fmt.Printf("Hello %s\n", context.Data["founder"])
```

The steps implementation might store their own context-specific data in
the `Data` map:
```go
  context.Data["result"] = "Fantastic!"
```


## Comparaison with other parallel cucumber solutions

In paracuke, the atomic unit is the test step (`Given`, `When`,
or `Then`). This means that the context might change between two steps
of the same scenario, unlike parallel-cucumber-js:

  https://github.com/simondean/parallel-cucumber-js

where the atomic unit is the scenario.

The parallelism in paracuke is acheived through coroutines, whereas
parallel-cucumber-js uses preemptive multitasking workers.
