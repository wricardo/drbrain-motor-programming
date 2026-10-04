# Dr Brain - Motor Programming

A puzzle game about programming a little robot, playable by **humans and AI alike**.

**▶ Play it now: https://motor-programming.wricardo.net/**

No sign-up, no install. Pick a map and start.

People play in the browser by clicking and dragging instructions. AI agents play the same game through a public **GraphQL API**: they read the map, write a program, run it and check the result. A human and an AI can even play the same session, and anyone can watch it live.

The project is also an experiment in using AI for something other than chat or code: here the AI has to solve a puzzle by itself, using an API it has never seen. See [Playing with AI](#playing-with-ai).

## How it works

A robot stands on a grid with rocks (`#`) and treats (`*`). Your job: get it to pick up every treat.

You don't steer the robot directly. You write it a short **program**, press **Run**, and watch it follow your instructions one step at a time. If it collects every treat, you win. If it runs out of instructions or hits the step limit first, you lose. Then you fix your program and try again.

A program is built from four kinds of instructions:

| Instruction | What the robot does |
|---|---|
| **Move forward** | Steps one square in the direction it's facing. Walking into a rock or off the edge does nothing. |
| **Turn left** | Turns 90° to the left, staying on its square. |
| **Turn right** | Turns 90° to the right, staying on its square. |
| **Call sub 1 / 2 / 3** | Runs one of your subroutines, then continues where it left off. |

## The catch: you don't have much room

Your main program has only a few slots, often far fewer than the moves the route needs. To fit, you need **subroutines**: small extra programs you can call by name.

If the robot has to repeat "move, move, turn right" four times, put that pattern in Sub 1 and call it four times. Subroutines can call other subroutines, so a few slots can turn into dozens of moves. On some maps a subroutine can even call **itself**, which gives you a loop.

Most of the game is spotting the pattern in the map and folding it into as few instructions as possible.

## Maps

There are 11 maps, ordered from easy to hard:

- **Levels 1–5** teach the basics: moving, turning, dodging rocks and your first subroutines.
- **The harder maps** are designed so the brute-force route doesn't fit. You have to find the trick: nested subroutines, subroutines that triple each other, or a subroutine that calls itself.

Every map can be solved. Each one has been checked against a known solution.

## More to do

- **Watch** anyone's game live from the sessions page, and see how other people solved a map.
- **Learn** more on the in-game [how it works](https://motor-programming.wricardo.net/learn) page: rules, strategies and tips.

## Playing with AI

There are two ways to get an AI to play.

**1. Copy the prompt from the game (easiest).** Start a game in the browser and click **Play with an AI**. It copies a ready-made prompt with the session, the map and the API calls the AI needs. Paste it into an AI assistant that can make web requests (for example a coding agent like Claude Code). Then open the game's watch page and see if the AI can win.

**2. Point an agent at the API.** The game publishes a guide written for AI agents at [/llms.txt](https://motor-programming.wricardo.net/llms.txt). It has the rules, the endpoints and examples for every call. Give your agent this link and ask it to beat a map.

The API is plain GraphQL at `https://motor-programming.wricardo.net/graphql`. A game takes four calls:

```graphql
query    { maps { id name difficulty layout mainTapeLength subTapeLengths } }   # 1. pick a map
mutation { createSession(mapID: "straight_line") { id } }                       # 2. start a game
mutation { setProgram(sessionID: "…", program: { main: [MOVE_FORWARD, …], subs: [[…]] }) { id } }  # 3. load a program
mutation { run(sessionID: "…") { id status } }                                  # 4. press Run
```

Then read the session (or subscribe to it) until the status is `WON` or `LOST`. If it lost, fix the program and try again, just like a human player.

Some questions worth exploring:

- Can the AI find the pattern on the hard maps, or does it only manage the easy ones?
- Does it use the `simulate` query to test a program before running it, or does it learn by losing?
- How many tries does it need compared to you?

## About

Motor Programming is the first game in **Dr Brain**, a collection of browser brain games. It's free to play.

Want to run it yourself or contribute? See [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md).
