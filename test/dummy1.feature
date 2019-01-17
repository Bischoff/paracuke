# Dummy feature 1/2

Feature: Random waits and arithmetic
  This is just to get an idea and to test paracuke

  Scenario: Express our satisfaction
    When I say "Yippee!"
    And I wait for a random time
    And I say "Hurray!"

  Scenario: Basic arithmetic
    When I add 2 and 3
    Then I should get 5
    # Intentionally failing
    When I wait for a random time
    And I add 1 and 1
    Then I should get 99
