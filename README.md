# Dr Brain - Motor Programming

A puzzle game about programming a little robot.

**▶ Play it now: https://motor-programming.wricardo.net/**

No sign-up, no install. Pick a map and start.

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
- **Play with an AI.** Each game has a "Play with an AI" button that copies a ready-made prompt. Paste it into an AI chat assistant and see if it can solve the map.
- **Learn** more on the in-game [how it works](https://motor-programming.wricardo.net/learn) page: rules, strategies and tips.

## About

Motor Programming is the first game in **Dr Brain**, a collection of browser brain games. It's free to play.

Want to run it yourself or contribute? See [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md).
