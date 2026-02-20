# Workflow Definition

## Session Protocol

### On Start

1. Read `.claudemod/SESSION_STATE.json`
2. Read `.claudemod/FIX_PLAN.md`
3. Read `.claudemod/CHANGELOG.md`
4. Read `.claudemod/spec/INDEX.md` and other specs in the `.claudemod/spec` folder
5. Drift detection: compare spec against code, flag discrepancies

In reading SESSION_STATE.json, determine the action to take based on the action field. 
If the action is "advance", then the phase listed in the "phase" field should be started.
If the action is "restart", then the phase listed in the "phase" field has not been completed and that phase should be restarted with the "discussion_summary" field as context and "recommendation" field as the next steps.
If the action is "rollback", then the phase listed in the "phase" field should be started with the "discussion_summary" field as context, "explanation" field as the reason for the rollback, and "recommendation" field as the next steps.
If the action is "complete", then the session is complete and the developer should be prompted to restart the workflow.

### On Phase Exit

Before signaling phase completion, update `.claudemod/SESSION_STATE.json` with the following JSON structure:

```
{
  "action": "advance",
  "discussion_summary": <3-4 line summary of discussion>,
  "recommendation": <2-3 line summary of what to focus on>
}
```

### On End

If work is incomplete, write `.claudemod/SESSION_STATE.json` with:

```
{
  "action": "restart",
  "phase": <the name of the current incomplete phase>,
  "discussion_summary": <3-4 line summary of discussion>,
  "recommendation": <2-3 line summary of what to focus on>
}
```

If work is complete, write `.claudemod/SESSION_STATE.json` with:

```
{
  "action": "complete"
}
```

## Rollback

Phases can signal a rollback to revisit an earlier phase when concrete problems are discovered.

### When to Rollback

- Requirements were misunderstood and need re-discussion
- Spec gaps discovered during implementation
- Wrong test assumptions that invalidate the test approach
- Architectural issues that require rethinking the design

### When NOT to Rollback

- Minor fixable issues (fix in the current phase instead)
- Speculative concerns without concrete evidence
- Small test adjustments that can be handled inline

### How to Signal Rollback

1. In interactive mode, discuss with the developer and get agreement first
2. Make sure only to rollback to valid rollback targets (valid targets are listed in each phase prompt)
3. Update `.claudemod/SESSION_STATE.json` with the following JSON structure:

```
{
  "action": "rollback",
  "phase": <the name of the phase>,
  "discussion_summary": <3-4 line summary of discussion>,
  "explanation": <2-3 line summary of why rollback is needed>,
  "recommendation": <2-3 line summary of what to focus on>,
}
```


## Phases

### discuss-feature

Before beginning, ask the developer to consider committing any prior changes to the project to source control before continuing.

Follow these steps:

1. Read the project spec.
2. Ask the developer clarifying questions about their request. After each response, state your confidence level (0-100%) in understanding what the developer wants to build. Continue asking questions until you reach at least 85% confidence.
3. After reaching >85% confidence, ask the developer to continue refining or proceed with sensible defaults.
4. After confirmation, end with a summary of understood requirements.

## Criteria

- [ ] Requirements summarized
- [ ] Developer confirms understanding

### spec-feature

Follow these steps:

1. Determine which spec files to create or update. Read related specs and check for contradictions. 
2. Draft changes covering data structures, architecture decisions, data models, API contracts behavioral requirements. Use `.claudemod/refs/SPEC.md` as a template to generate the spec.
3. Describe the plan to the developer. Start with talking about the abstraction model and the data structures involved and, if applicable, separate the discussion into the different parts of the system (for example, the frontend, backend, database, different microservices, etc.). Here are some guidelines for describing the plan:

Show the interfaces in this format:
```
type MyNewInterface interface {
	DoSomething() error
	AnotherMethod() (string, error)
  DoSomethingElse() (int, error)
}
```

Show the data structures in this format:
```
// MyNewStruct is a new struct that represents a new data structure.
// Implements the MyNewInterface interface.
type MyNewStruct struct {
	Name string `json:"name"`
	Age  int    `json:"age,omitempty"`

  somePrivateField      string
  anotherPrivateField   map[string]string
  myCoolChannel         <-chan string
  myCoolMutex           sync.Mutex
}
```

Talk about the functions needed and how it connects to the greater whole.
When talking about the system as a whole, if a user or event is involved trace the execution paths from start to finish.
If possible, draw a simplified diagram of the data structures and how they connect to each other and the user or event in this format:
```
                          +-----------------------+
            +-----------> |   Payment Processor   | ----+
            |             +-----------------------+     |
            |                                           |
        route request                           confirms payment
    to payment processor                                |
            |                                           v
+-----------------------+   submits order   +-----------------------+
|   HTTP POST /payment  | ----------------> |     Order Processor   |
+-----------------------+ <---------------- +-----------------------+
                           confirms order                           
```

4. Ask the developer to confirm the spec changes.

## Criteria

- [ ] Spec changes drafted
- [ ] No unresolved contradictions
- [ ] Changes written to .claudemod/spec files
- [ ] Developer confirms changes to the spec

### scope-feature

Follow these steps:

1. Break the agreed work into concrete, implementable tasks.
2. Write tasks to `.claudemod/FIX_PLAN.md` with checkboxes.
3. Each task should be small enough to implement and test in one iteration.
4. Order the tasks from high priority, medium priority, and low priority.
5. Ask the developer to confirm the tasks.

## Criteria

- [ ] .claudemod/FIX_PLAN.md populated with tasks ordered by priority
- [ ] Each task is concrete and testable
- [ ] Developer confirms tasks

### tdd-red

Write test cases for all the tasks in the `.claudemod/FIX_PLAN.md` file.
For each task:

1. Define the interface/contract using language-appropriate constructs. Examples include:

- Go: interface
- TypeScript: interface/type
- Python: Protocol/ABC,
- Rust: trait

2. Determine the test cases for the interfaces, such as nil-/null- pointer issues, negative indexes, or default values.
3. Construct tooling for the tests, such as mocks, stubs, or test harnesses and fixtures.
4. Write tests against the interface
5. Run the tests — they MUST fail due to the interface not being implemented, not due to other errors such as syntax or linting

Once the tests are designed and written, summarize the test cases and explain the test tooling that was created. Then ask the developer to confirm the tests are correct and complete. Be sure to explain why the interface is necessary for the task and, if applicable, the reasoning behind the test cases in detail and why they are important. If the developer rejects the test cases, ask them to explain why and work with them to design new test cases that are acceptable to them. Once the developer confirms the test cases, make sure they fail due to the interface not being implemented, not due to other errors such as syntax or linting.

## Criteria

- [ ] Interface/contract defined
- [ ] Test tooling constructed
- [ ] Tests written
- [ ] Developer confirms tests
- [ ] Tests run and FAIL (red)

### tdd-green

Execute the following steps in a loop until all tests pass:

1. Write the minimal implementation to make tests pass. Follow these guidelines during implementation:

- Run the tests after every implementation change
- Do not add functionality beyond what tests require.
- If new data structures or modules are written, be sure to explain why the interface is necessary for the task and, if applicable, the reasoning behind the design decision in concise detail and why it was important. If the developer rejects the implementation, ask them to explain why and work with them to design a new or revised implementation that is acceptable to them.

2. Ask the developer to confirm the implementation. After confirmation, tick the task checkboxes as completed in `.claudemod/FIX_PLAN.md`.
3. Check that the implementation fulfills the task requirements.

After completing the loop, check if there are any tasks not completed in `.claudemod/FIX_PLAN.md`. If there are, suggest a rollback to the tdd-red phase to write tests for the missing functionality. Measure the code coverage of the tests and ensure it is at least 80% otherwise suggest a rollback to the tdd-red phase to write more tests.

## Criteria

- [ ] Implementation written
- [ ] All tests PASS (green)
- [ ] Developer confirms implementation

### code-review

Execute the following steps in a loop until the code review yields either only low priority issues or no issues.

1. Review the design of the implementation. Check for the following:

- errors
- code coverage (at least 80%)
- input validation
- no hardcoded secrets
- immutable patterns
- function size (<50 lines)
- file size (<500 lines)

Here are some general design principles that should be followed:

- Errors should be handled explicitly.
- Input validation should be done explicitly.
- No hardcoded secrets should be used.
- Immutable patterns should be used.
- Function size should be less than 50 lines. If a function is too large, it should be split into smaller functions.
- Functions should be small and focused on a single responsibility. This could be a small task or the coordination of function calls that multiple related tasks.
- File size should be less than 500 lines. If a file is too large, it should be split into smaller files.
- Code for similar functionality inside a domain should be co-located. For example, if you are writing a function to
handle a user login, other login-related functionality should be written in the same file, package, or folder.
- Related classes and data structures should be written in the same file, unless the file is already too large.
- Polymorphism should be handled in the same file, unless the file is too large in which case it should be kept to the same package or folder.
- Dependencies for classes or data structures should be injected via constructor or factory methods, not instantiated within the class or data structure.
- Data structures should be immutable unless they are mutable by design.
- Classes or data structures should be extensible without modifying the existing code.

2. If the code does not follow these principles, suggest a refactoring plan and ask the developer to confirm it. Structure the refactoring plan as a list of tasks to be completed with critical priority, high priority, medium priority, and low priority.
3. After confirmation, write any refactoring tasks that need to be completed to `.claudemod/FIX_PLAN.md` with checkboxes.
4. Then execute the refactoring tasks in order of priority, starting with the critical priority tasks. For each refactoring task:

- write the minimal implementation to make the tests pass
- run the tests and confirm they pass (green)
- ask the developer to confirm the implementation
- if the developer rejects the refactoring task, ask them to explain why and work with them to design a new or revised implementation that is acceptable to them. Once the developer confirms the implementation, tick the task checkbox as completed in `.claudemod/FIX_PLAN.md`.

**Criteria**

- [ ] Code review
- [ ] Refactoring tasks written to .claudemod/FIX_PLAN.md
- [ ] Developer confirms review
- [ ] All refactoring tasks completed

### synthesize-specs

Complete the following steps:

1. Re-read entire `.claudemod/spec/` directory.
2. Compare each spec against the implementation.
3. Update specs to reflect code as built.
4. Check for gaps (code not in spec) and phantom specs (spec not in code).
5. Append dated entry to `.claudemod/CHANGELOG.md` with a summary of the changes to the specs and the implementation.
6. Copy completed tasks from `.claudemod/FIX_PLAN.md` to `.claudemod/CHANGELOG.md`.
7. Remove completed tasks from `.claudemod/FIX_PLAN.md`.

Then, complete these steps:

1. Either get the changes from the last source control commit or from the last time the specs were synthesized.
2. Teach the developer about the changes to the specs and the code implementation. Have a conversation with the developer about the changes to the specs and the implementation, like a senior developer would to a junior developer. Use diagrams and code examples to help explain the changes. If the developer does not understand the changes, ask them to explain why and work with them to understand the changes. Once the developer understands the changes, ask them to confirm they understand the changes.
3. Ask the developer to confirm they understand the changes. If they do not understand the changes, ask them to explain why and work with them to understand the changes. Once the developer understands the changes, ask them to confirm they understand the changes.

## Criteria

- [ ] Specs updated to reflect code as built
- [ ] Gaps and phantom specs identified
- [ ] CHANGELOG.md updated
- [ ] FIX_PLAN.md cleared
- [ ] Developer understands the changes to the specs and the implementation

### bootstrap

Explore the codebase and look for the following things:

- code structure
- architecture
- languages
- frameworks
- design patterns
- dependencies
- major domains

Ask as many questions as you need to be at least 95% sure that you understand:

- the purpose of the application
- the application's intended users
- any key features of this application
- how it fits into a larger system, if applicable
- architecture decisions

1. Use `.claudemod/refs/SPEC_INDEX.md` as a template and create an `INDEX.md` that summarizes the information gathered in the `.claudemod/spec` folder.

2. Generate specs for each identified domain into that folder with the `{domain}/{subdomain}.MD` filename pattern and use `.claudemod/refs/SPEC.md` as a template to generate these specs and write them to the `.claudemod/spec` folder and link them in the `INDEX.md` file.

3. Also generate the following files:
- a blank `.claudemod/SESSION_STATE.json` file
- a blank `.claudemod/FIX_PLAN.md` file

## Criteria

- [ ] .claudemod/spec/INDEX.md populated
- [ ] At least one domain spec created and linked to INDEX.md
- [ ] Developer reviewed and accepted
