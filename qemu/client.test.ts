import { afterEach, describe, expect, test } from "bun:test";
import { mkdtempSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { assert, newDisk, newQemu, screendump, start, stop, type Qemu } from "./client";

const dirs: string[] = [];
const running: Qemu[] = [];

function scratch(): string {
	const dir = mkdtempSync(join(tmpdir(), "qemu-client-"));
	dirs.push(dir);
	return dir;
}

async function started(): Promise<Qemu> {
	const vm = newQemu();
	await newDisk(vm, join(scratch(), "disk.qcow2"), "64M");
	await start(vm);
	running.push(vm);
	return vm;
}

afterEach(async () => {
	for (const vm of running.splice(0)) {
		if (vm.proc) await stop(vm);
	}
	for (const dir of dirs.splice(0)) {
		rmSync(dir, { recursive: true, force: true });
	}
});

describe("assert", () => {
	test("happy: a truthy value passes through without throwing", () => {
		expect(() => assert("disk.qcow2", "never thrown")).not.toThrow();
	});

	test("unhappy: a falsy value throws the message", () => {
		expect(() => assert(undefined, "the disk is missing")).toThrow(
			"the disk is missing",
		);
	});
});

describe("newDisk", () => {
	test("happy: the fresh vm has no disk path, and newDisk defines it with a real qcow2", async () => {
		const vm = newQemu();
		expect(vm.diskPath).toBeUndefined();

		const path = join(scratch(), "disk.qcow2");
		await newDisk(vm, path, "64M");

		expect(vm.diskPath).toBe(path);
		const magic = [...readFileSync(path).subarray(0, 4)];
		expect(magic).toEqual([0x51, 0x46, 0x49, 0xfb]); // "QFI\xfb"
	});

	test("unhappy: a second newDisk on the same vm throws", async () => {
		const vm = newQemu();
		await newDisk(vm, join(scratch(), "disk.qcow2"), "64M");

		await expect(
			newDisk(vm, join(scratch(), "again.qcow2"), "64M"),
		).rejects.toThrow("disk already created");
	});
});

describe("start", () => {
	test(
		"happy: start puts the live process and QMP socket on the vm",
		async () => {
			const vm = await started();

			expect(vm.proc?.pid).toBeGreaterThan(0);
			expect(vm.proc?.exitCode).toBeNull();
			expect(vm.qmp).toBeDefined();
		},
		{ timeout: 30_000 },
	);

	test("unhappy: start without a disk throws", async () => {
		const vm = newQemu();
		await expect(start(vm)).rejects.toThrow("no disk");
	});
});

describe("screendump", () => {
	test(
		"happy: a started vm dumps its screen as a PPM",
		async () => {
			const vm = await started();

			const shot = join(scratch(), "screen.ppm");
			await screendump(vm, shot);

			expect(readFileSync(shot).subarray(0, 2).toString()).toBe("P6");
		},
		{ timeout: 30_000 },
	);

	test("unhappy: screendump before start throws", async () => {
		const vm = newQemu();
		await expect(screendump(vm, join(scratch(), "screen.ppm"))).rejects.toThrow(
			"not started",
		);
	});
});

describe("stop", () => {
	test(
		"happy: stop ends the process and clears it off the vm",
		async () => {
			const vm = await started();
			const pid = vm.proc!.pid!;

			await stop(vm);

			expect(vm.proc).toBeUndefined();
			expect(vm.qmp).toBeUndefined();
			expect(() => process.kill(pid, 0)).toThrow();
		},
		{ timeout: 30_000 },
	);

	test("unhappy: stop before start throws", async () => {
		const vm = newQemu();
		await expect(stop(vm)).rejects.toThrow("not started");
	});
});
