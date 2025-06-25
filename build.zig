const std = @import("std");

const APP_NAME = "sbctl";
const VERSION = "0.1.0";
const SING_BOX_VERSION = "0.11.x";

pub fn build(b: *std.Build) !void {
    const alloc = b.allocator;
    const target = b.standardTargetOptions(.{});
    const mod_name = try getGoModuleName(alloc);

    var version = b.option([]const u8, "version", "Specify the program version") orelse VERSION;
    const sb_version = b.option([]const u8, "sing-box-version", "Specify supported sing-box version") orelse SING_BOX_VERSION;
    var config_home = b.option([]const u8, "config-home", "App home directory") orelse switch (target.result.os.tag) {
        .linux => "/etc/sing-box-ctl",
        .windows => "%LOCALAPPDATA%/sing-box-ctl",
        inline else => unreachable,
    };
    const prod = b.option(bool, "prod", "Build with production mode") orelse false;
    if (!prod) {
        config_home = b.pathJoin(&.{ b.install_prefix, "sing-box-ctl" });
        version = "dev";
    }

    const app_full_path = b.pathJoin(&.{ b.install_prefix, APP_NAME });
    const go_build = b.addSystemCommand(&.{ "go", "build", "-o", app_full_path });

    const go_ldflags = try std.mem.join(alloc, " ", &.{
        b.fmt("-X {s}/cmd.Version={s}", .{ mod_name, version }),
        b.fmt("-X {s}/cmd.SingBoxVersion={s}", .{ mod_name, sb_version }),
        b.fmt("-X {s}/config.ConfigHome={s}", .{ mod_name, config_home }),
        b.fmt("-X {s}/logger.Production={s}", .{ mod_name, if (prod) "prod" else "" }),
    });

    go_build.addArgs(&.{ "-ldflags", go_ldflags });
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

fn getGoModuleName(alloc: std.mem.Allocator) ![]const u8 {
    const mod_file = try std.fs.cwd().openFile("go.mod", .{});
    defer mod_file.close();
    var reader = std.io.bufferedReader(mod_file.reader());
    const r = reader.reader();
    const first_line = try r.readUntilDelimiterAlloc(alloc, '\n', 1024);
    return first_line[7..];
}

fn arch2go(arch: std.Target.Cpu.Arch) []const u8 {
    return switch (arch) {
        .x86_64 => "amd64",
        .aarch64 => "arm64",
        inline else => @panic("unsupported cpu arch"),
    };
}
