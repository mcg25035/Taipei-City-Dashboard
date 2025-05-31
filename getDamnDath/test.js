class ProgressBar {
  constructor(format, options) {
    this.format = format;
    this.total = options.total;
    this.width = options.width || 20;
    this.completeChar = options.complete || '=';
    this.incompleteChar = options.incomplete || ' ';
    this.curr = 0;
    this.complete = false;
  }

  tick() {
    this.curr++;
    if (this.curr >= this.total) {
      this.curr = this.total;
      this.complete = true;
    }
    this.render();
  }

  render() {
    const percent = this.curr / this.total;
    const completeLength = Math.round(this.width * percent);
    const incompleteLength = this.width - completeLength;
    const bar =
      this.completeChar.repeat(completeLength) +
      this.incompleteChar.repeat(incompleteLength);
    const percentStr = (percent * 100).toFixed(0) + '%';
    const eta = this.complete ? '0s' : Math.ceil((this.total - this.curr) * 0.1) + 's';
    process.stdout.clearLine();
    process.stdout.cursorTo(0);
    process.stdout.write(
      this.format
        .replace(':bar', bar)
        .replace(':percent', percentStr)
        .replace(':etas', eta)
    );
  }
}

// Create a new progress bar instance
const bar = new ProgressBar('Progress [:bar] :percent :etas', {
  total: 20,
  width: 30,
  complete: '=',
  incomplete: ' ',
});

// Simulate progress
const timer = setInterval(() => {
  bar.tick();
  if (bar.complete) {
    clearInterval(timer);
    console.log('\nDone!');
  }
}, 100);