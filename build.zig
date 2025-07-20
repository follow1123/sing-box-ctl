const std = @import("std");

const APP_NAME = "sbctl";

const Params = struct {
    b: *std.Build,
    dev_config_home: []const u8,
    prod: bool,
    app_full_path: []const u8,

    pub fn init(b: *std.Build) Params {
        const dev_config_home = b.option([]const u8, "config-home", "Set config home directory for development") orelse b.pathJoin(&.{ b.install_prefix, "dev-config-home" });
        const prod = b.option(bool, "prod", "Build with production mode") orelse false;
        return .{
            .b = b,
            .dev_config_home = dev_config_home,
            .prod = prod,
            .app_full_path = b.pathJoin(&.{ b.install_prefix, APP_NAME }),
        };
    }
};

const targets: []const std.Target.Query = &.{
    .{ .cpu_arch = .x86_64, .os_tag = .linux },
    .{ .cpu_arch = .x86_64, .os_tag = .windows },
};

pub fn build(b: *std.Build) !void {
    const target = b.standardTargetOptions(.{});

    var params = Params.init(b);

    const go_build = try goBuildStep(b, target, &params, null);
    b.getInstallStep().dependOn(&go_build.step);

    const go_run = try goRunStep(b, &params);
    go_run.step.dependOn(&go_build.step);

    const run_step = b.step("run", "Run the Go application");
    run_step.dependOn(&go_run.step);

    const targets_dir = "targets";

    const releases_step = b.step("releases", "Build all target releases");

    for (targets) |t| {
        const go_release_build = try goBuildStep(b, b.resolveTargetQuery(t), &params, targets_dir);
        releases_step.dependOn(&go_release_build.step);
    }
}

fn goRunStep(b: *std.Build, params: *Params) !*std.Build.Step.Run {
    const go_run = b.addSystemCommand(&.{params.app_full_path});

    if (b.args) |args| go_run.addArgs(args);
    return go_run;
}

fn goBuildStep(b: *std.Build, target: std.Build.ResolvedTarget, params: *Params, release_targets_dir: ?[]const u8) !*std.Build.Step.Run {
    const root_file_path = try b.build_root.join(b.allocator, &.{"go.mod"});
    const mod_name = try getGoModuleName(b.allocator, root_file_path);

    var prod = params.prod;
    var app_full_path: []const u8 = undefined;
    if (release_targets_dir) |targets_dir| {
        prod = true;
        const target_path = try target.query.zigTriple(b.allocator);
        app_full_path = b.pathJoin(&.{ b.install_prefix, targets_dir, target_path, APP_NAME });
    } else {
        app_full_path = params.app_full_path;
    }

    const go_build = b.addSystemCommand(&.{ "go", "build", "-o", app_full_path });

    var ldflags = std.ArrayList([]const u8).init(b.allocator);
    try ldflags.append(b.fmt("-X {s}/logger.Production={s}", .{ mod_name, if (prod) "true" else "false" }));
    if (!prod) {
        try ldflags.append(b.fmt("-X {s}/config.ConfigHome={s}", .{ mod_name, params.dev_config_home }));
    }

    const go_ldflags = try std.mem.join(b.allocator, " ", ldflags.items);
    go_build.addArgs(&.{ "-ldflags", go_ldflags });
    go_build.setEnvironmentVariable("CGO_ENABLED", "0");
    go_build.setEnvironmentVariable("GOARCH", arch2go(target.result.cpu.arch));
    go_build.setEnvironmentVariable("GOOS", @tagName(target.result.os.tag));
    return go_build;
}

fn getGoModuleName(alloc: std.mem.Allocator, mod_file_path: []const u8) ![]const u8 {
    const mod_file = try std.fs.openFileAbsolute(mod_file_path, .{});
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
