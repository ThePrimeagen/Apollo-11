import { execFile, spawn, type ChildProcess } from "node:child_process";
import { createConnection, type Socket } from "node:net";
import { dirname, join } from "node:path";
import { promisify } from "node:util";

const exec = promisify(execFile);

export type Qemu = {
	diskPath: string | undefined;
	proc: ChildProcess | undefined;
	qmp: Socket | undefined;
};

export function assert(value: unknown, message: string): asserts value {
	if (!value) {
		throw new Error(message);
	}
}

export function newQemu(): Qemu {
	return { diskPath: undefined, proc: undefined, qmp: undefined };
}

export async function newDisk(
	vm: Qemu,
	path: string,
	size: string,
): Promise<void> {
	assert(vm.diskPath === undefined, "disk already created");
	await exec("qemu-img", ["create", "-f", "qcow2", path, size]);
	vm.diskPath = path;
}

export async function start(vm: Qemu): Promise<void> {
	assert(vm.diskPath !== undefined, "no disk");
	const socketPath = join(dirname(vm.diskPath), "qmp.sock");
	vm.proc = spawn(
		"qemu-system-x86_64",
		[
			"-display", "none",
			"-m", "128",
			"-drive", `file=${vm.diskPath},format=qcow2,if=virtio`,
			"-qmp", `unix:${socketPath},server=on,wait=off`,
		],
		{ stdio: "ignore" },
	);
	vm.qmp = await connect(socketPath);
	await command(vm.qmp, "qmp_capabilities");
}

export async function screendump(vm: Qemu, path: string): Promise<void> {
	assert(vm.qmp, "not started");
	await command(vm.qmp, "screendump", { filename: path });
}

export async function stop(vm: Qemu): Promise<void> {
	assert(vm.proc, "not started");
	const proc = vm.proc;
	const gone = new Promise<void>((resolve) => proc.once("exit", () => resolve()));
	vm.qmp?.destroy();
	proc.kill();
	await gone;
	vm.proc = undefined;
	vm.qmp = undefined;
}

type QmpMessage = {
	return?: unknown;
	error?: { desc: string };
};

function connect(path: string): Promise<Socket> {
	return new Promise((resolve) => {
		const attempt = () => {
			const socket = createConnection(path);
			socket.once("connect", () => {
				socket.removeAllListeners("error");
				socket.on("error", () => {});
				resolve(socket);
			});
			socket.once("error", () => setTimeout(attempt, 100));
		};
		attempt();
	});
}

function command(
	qmp: Socket,
	execute: string,
	args?: Record<string, unknown>,
): Promise<void> {
	return new Promise((resolve, reject) => {
		let buffered = "";
		const onData = (chunk: Buffer) => {
			buffered += chunk.toString();
			const lines = buffered.split("\n");
			buffered = lines.pop() ?? "";
			for (const line of lines) {
				if (!line.trim()) {
					continue;
				}
				const message = JSON.parse(line) as QmpMessage;
				if ("return" in message) {
					qmp.off("data", onData);
					resolve();
					return;
				}
				if (message.error) {
					qmp.off("data", onData);
					reject(new Error(message.error.desc));
					return;
				}
			}
		};
		qmp.on("data", onData);
		qmp.write(`${JSON.stringify({ execute, arguments: args })}\n`);
	});
}
