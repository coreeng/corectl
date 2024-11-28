Feature: Greeting
  It's polite to greet someone

  Scenario: hello world returns ok
    Given a rest service
    When I call the hello world endpoint
    Then an ok response is returned

  Scenario: ingress url hello world returns ok
    Given a rest service
    When I call the ingress hello world endpoint and wait for it to be ready
    Then an ok response is returned