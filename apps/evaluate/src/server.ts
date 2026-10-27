// The evaluate service over HTTP, for the Mac (S41; A-22, local first): `npm run evaluate`, which
// `make dev` starts beside the API and the web app. It listens on 127.0.0.1 only.
import { createServer, type Server } from "node:http";
import { handle, MAX_BODY } from "./handler";

export function startServer({ port, version }: { port: number; version: string }): Promise<Server> {
  const srv = createServer((req, res) => {
    const chunks: Buffer[] = [];
    let size = 0;
    let tooBig = false;
    req.on("data", (c: Buffer) => {
      size += c.length;
      if (size > MAX_BODY) tooBig = true;
      else chunks.push(c);
    });
    req.on("end", async () => {
      const reply = tooBig
        ? { status: 413, body: { error: "The body is too big." } }
        : await handle(req.method ?? "", new URL(req.url ?? "/", "http://x").pathname, Buffer.concat(chunks).toString("utf8"), version);
      const text = JSON.stringify(reply.body);
      res.writeHead(reply.status, { "Content-Type": "application/json", "Content-Length": Buffer.byteLength(text) });
      res.end(text);
    });
  });
  return new Promise((ready, fail) => {
    srv.once("error", fail);
    srv.listen(port, "127.0.0.1", () => ready(srv));
  });
}
