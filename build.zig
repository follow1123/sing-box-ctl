const std = @import("std");

const app_name = "sbctl";

pub fn build(b: *std.Build) !void {
    const app_full_path = std.fs.path.join(b.allocator, &.{ b.exe_dir, app_name }) catch @panic("OOM");
    const target = b.standardTargetOptions(.{});
    const go_build = b.addSystemCommand(&.{ "go", "build", "-o", app_full_path });
    go_build.setEnvironmentVariable("CGO_ENABLED", "0");
    go_build.setEnvironmentVariable("GOARCH", arch2go(target.result.cpu.arch));
    go_build.setEnvironmentVariable("GOOS", @tagName(target.result.os.tag));

    b.getInstallStep().dependOn(&go_build.step);

    const go_run = b.addSystemCommand(&.{app_full_path});

    if (b.args) |args| go_run.addArgs(args);

    go_run.step.dependOn(&go_build.step);

    const run_step = b.step("run", "Run the Go application");
    run_step.dependOn(&go_run.step);
}


fn arch2go(arch: std.Target.Cpu.Arch) []const u8 {
    return switch (arch) {
        .x86_64 => "amd64",
        .aarch64 => "arm64",
        inline else => @panic("unsupported cpu arch"),
    };
}
