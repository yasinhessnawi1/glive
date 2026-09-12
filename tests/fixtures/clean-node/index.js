// Deliberately free of every scanner pattern: no eval, no exec, no
// child_process, no credentials, no shell pipes.
const values = [1, 2, 3];

function total(items) {
  return items.reduce((acc, n) => acc + n, 0);
}

console.log("total:", total(values));
