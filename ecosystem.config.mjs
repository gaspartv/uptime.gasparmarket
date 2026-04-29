// https://pm2.keymetrics.io/docs/usage/application-declaration/

import { dirname } from "node:path";
import { fileURLToPath } from "node:url";

const rootDir = dirname(fileURLToPath(import.meta.url));

export const apps = [
  {
    name: "gasparmarket[uptime]",
    cwd: rootDir,
    script: "./uptime.gasparmarket",
    interpreter: "none",
    watch: false,
    autorestart: true,
    max_memory_restart: "800M",
  },
];

export default { apps };
