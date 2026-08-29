import { describe, expect, test } from "bun:test";

import { assert, newDisk, newQemu, screendump, start, stop } from "./client";

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

describe("the qemu object", () => {
	test("happy: a fresh vm has nothing defined yet", () => {
		const vm = newQemu();
		expect(vm.diskPath).toBeUndefined();
		expect(vm.proc).toBeUndefined();
		expect(vm.qmp).toBeUndefined();
	});

	test("unhappy: newDisk refuses a vm whose disk is already created", async () => {
		const vm = newQemu();
		vm.diskPath = "already.qcow2";
		await expect(newDisk(vm, "again.qcow2", "64M")).rejects.toThrow(
			"disk already created",
		);
	});

	test("unhappy: start refuses a vm with no disk", async () => {
		await expect(start(newQemu())).rejects.toThrow("no disk");
	});

	test("unhappy: screendump refuses a vm that never started", async () => {
		await expect(
			screendump(newQemu(), "screen.ppm"),
		).rejects.toThrow("not started");
	});

	test("unhappy: stop refuses a vm that never started", async () => {
		await expect(stop(newQemu())).rejects.toThrow("not started");
	});
});
