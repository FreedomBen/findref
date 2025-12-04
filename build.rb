#!/usr/bin/env ruby

#
# This is not meant to be an entrypoint.  Use `rake` with the rakefile
#

require 'fileutils'
require 'erb'

require_relative 'helpers'

GO_VERSION = '1.25'.freeze
WORKDIR_IN_CONTAINER = '/usr/src/findref'.freeze
NFPM_IMAGE = 'goreleaser/nfpm:v2.37.0'.freeze
NFPM_CONFIG_TEMPLATE = 'packaging/nfpm.yaml.erb'.freeze
NFPM_GENERATED_CONFIG = 'dist/nfpm.generated.yaml'.freeze
LINUX_PACKAGE_FORMATS = {
  'deb' => { extension: 'deb' },
  'rpm' => { extension: 'rpm' },
  'apk' => { extension: 'apk' },
  'archlinux' => { extension: 'pkg.tar.zst', arches: %w(amd64) }
}.freeze
NFPM_ARCH_MAP = {
  'deb' => {
    'amd64' => 'amd64',
    '386' => 'i386',
    'arm' => 'armhf',
    'arm64' => 'arm64'
  },
  'rpm' => {
    'amd64' => 'x86_64',
    '386' => 'i386',
    'arm' => 'armv7hl',
    'arm64' => 'aarch64'
  },
  'apk' => {
    'amd64' => 'x86_64',
    '386' => 'x86',
    'arm' => 'armhf',
    'arm64' => 'aarch64'
  },
  'archlinux' => {
    'amd64' => 'x86_64'
  }
}.freeze

# See: https://stackoverflow.com/a/30068222/2062384 for list of valid targets
OSES_ARCHES = {
  'linux' => %w(amd64 386 arm arm64),
  'windows' => %w(amd64 386),
  'darwin' => %w(amd64 arm64),
  'freebsd' => %w(amd64 arm64),
  'openbsd' => %w(amd64 arm64)
}.freeze

RED = "\033[0;31m".freeze
BLUE = "\033[0;34m".freeze
CYAN = "\033[0;36m".freeze
GREEN = "\033[0;32m".freeze
BROWN = "\033[0;33m".freeze
BLACK = "\033[0;30m".freeze
WHITE = "\033[1;37m".freeze
PURPLE = "\033[0;35m".freeze
YELLOW = "\033[1;33m".freeze
RESTORE = "\033[0m".freeze
LIGHT_RED = "\033[1;31m".freeze
DARK_GRAY = "\033[1;30m".freeze
LIGHT_GRAY = "\033[0;37m".freeze
LIGHT_BLUE = "\033[1;34m".freeze
LIGHT_CYAN = "\033[1;36m".freeze
LIGHT_GREEN = "\033[1;32m".freeze
LIGHT_PURPLE = "\033[1;35m".freeze

def cyan(message)
  puts "#{CYAN}#{message}#{RESTORE}"
end

def die(message)
  puts "#{RED}[die]: #{message}#{RESTORE}"
  exit 1
end

def podman_or_docker
  # If podman is installed, use that. Otherwise use docker
  `command -v podman`.strip.empty? \
    ? 'docker' \
    : 'sudo podman'
end

def docker_run(os, arch)
  <<-EOS.gsub(/\s+/, ' ').gsub(/[\s\t]*\n/, ' ').strip
    #{podman_or_docker} run
    --rm
    --volume "#{Dir.pwd}:#{WORKDIR_IN_CONTAINER}:Z"
    --workdir "#{WORKDIR_IN_CONTAINER}"
    --env GOOS=#{os}
    --env GOARCH=#{arch}
    golang:#{GO_VERSION} go build -buildvcs=false
  EOS
end

def stage_dir_rel(_os = nil, _arch = nil)
  File.join('dist', 'pkgroot')
end

def packages_dir_rel(os, arch)
  File.join('dist', 'packages', os, arch)
end

def container_path(relative_path)
  "#{WORKDIR_IN_CONTAINER}/#{relative_path}"
end

def package_formats_for_arch(arch)
  LINUX_PACKAGE_FORMATS.select do |_, spec|
    spec[:arches].nil? || spec[:arches].include?(arch)
  end
end

def package_filename(packager, extension, release, arch, package_arch)
  arch_suffix = package_arch || arch
  case packager
  when 'deb'
    "findref_#{release}_#{arch_suffix}.#{extension}"
  when 'rpm'
    "findref-#{release}-1.#{arch_suffix}.#{extension}"
  when 'apk'
    "findref-#{release}.#{arch_suffix}.#{extension}"
  when 'archlinux'
    "findref-#{release}-1-#{arch_suffix}.#{extension}"
  else
    "findref-#{release}-#{arch_suffix}.#{extension}"
  end
end

def ensure_nfpm_template!
  die("nfpm config template '#{NFPM_CONFIG_TEMPLATE}' not found") unless File.exist?(NFPM_CONFIG_TEMPLATE)
end

def render_nfpm_config(release, package_arch)
  ensure_nfpm_template!
  template = ERB.new(File.read(NFPM_CONFIG_TEMPLATE))
  rendered = template.result_with_hash(version: release, arch: package_arch)
  FileUtils.mkdir_p(File.dirname(NFPM_GENERATED_CONFIG))
  File.write(NFPM_GENERATED_CONFIG, rendered)
  NFPM_GENERATED_CONFIG
end

def nfpm_arch_for(packager, arch)
  mapping = NFPM_ARCH_MAP[packager]
  return arch if mapping.nil?

  mapping[arch] || arch
end

def nfpm_command(packager, target_rel, release, package_arch, config_rel)
  target_container = container_path(target_rel)
  config_container = container_path(config_rel)
  <<-EOS.gsub(/\s+/, ' ').gsub(/[\s\t]*\n/, ' ').strip
    #{podman_or_docker} run
    --rm
    --volume "#{Dir.pwd}:#{WORKDIR_IN_CONTAINER}:Z"
    --workdir "#{WORKDIR_IN_CONTAINER}"
    --env NFPM_VERSION=#{release}
    --env NFPM_ARCH=#{package_arch}
    #{NFPM_IMAGE}
    pkg --packager #{packager}
    --config #{config_container}
    --target #{target_container}
  EOS
end

def stage_linux_binary(fr_bin, arch)
  stage_rel = stage_dir_rel('linux', arch)
  stage_abs = File.join(Dir.pwd, stage_rel)
  FileUtils.rm_rf(stage_abs)
  FileUtils.mkdir_p(File.join(stage_abs, 'usr', 'bin'))
  staged_bin = File.join(stage_abs, 'usr', 'bin', 'findref')
  fr_bin_path = File.join(Dir.pwd, fr_bin)
  FileUtils.cp(fr_bin_path, staged_bin)
  FileUtils.chmod(0o755, staged_bin)
  stage_rel
end

def build_linux_packages(fr_bin, release, arch)
  stage_rel = stage_linux_binary(fr_bin, arch)
  artifacts = []
  package_formats_for_arch(arch).each do |packager, spec|
    package_arch = nfpm_arch_for(packager, arch)
    target_rel = File.join(packages_dir_rel('linux', arch), package_filename(packager, spec[:extension], release, arch, package_arch))
    target_abs = File.join(Dir.pwd, target_rel)
    FileUtils.mkdir_p(File.dirname(target_abs))
    cyan "Packaging #{packager} for linux #{arch}..."
    config_rel = render_nfpm_config(release, package_arch)
    command = nfpm_command(packager, target_rel, release, package_arch, config_rel)
    system(command) || die("nfpm packager '#{packager}' failed for linux #{arch}")
    artifacts << target_abs
  end
  FileUtils.rm_rf(File.join(Dir.pwd, stage_rel))
  FileUtils.rm_f(File.join(Dir.pwd, NFPM_GENERATED_CONFIG))
  artifacts
end

#
# Consider moving to github releases instead of a separate bin repo.
#   See: https://github.com/aktau/github-release/ for a useful tool to automate.
#   Example usage:  https://github.com/zyedidia/micro/blob/master/tools/release.sh
#
# Also consider moving to a Makefile.  See:
#   https://github.com/aktau/github-release/blob/master/Makefile and
#   https://github.com/zyedidia/micro for examples.
#
# The Micro source also has a snapcraft config to look at, as well as a brew package:
#   https://github.com/zyedidia/micro/blob/master/snapcraft.yaml
#
# Move build scripts and stuff to subdir
#
# Add config file (local in proj or global, just walk the tree up looking for it)
#
# Add option to honor git ignore file (currently excludes based on hidden files)
#
def main(release)
  die('Must pass release version as first arg') if release.nil? || release.empty?
  OSES_ARCHES.each do |os, arches|
    arches.each do |arch|
      dest_dirs = [Helpers.latest_release_name, release].map{ |rel| "#{Helpers.findref_bin_repo}/#{rel}/#{os}/#{arch}" }
      cyan "Building findref v#{release} for #{os} #{arch}..."
      cyan "Running: #{docker_run(os, arch)}"
      system(docker_run(os, arch))
      fr_bin = Helpers.bin_name(os)
      fr_zip = Helpers.zip_name
      package_files = []
      if os == 'linux'
        package_files = build_linux_packages(fr_bin, release, arch)
      end
      cyan "Zipping #{fr_bin} into #{fr_zip}"
      system("zip -9 #{fr_zip} #{fr_bin}")
      dest_dirs.each do |dest_dir|
        FileUtils.mkdir_p(dest_dir)
        FileUtils.cp(fr_zip, "#{dest_dir}/")
        package_files.each do |pkg|
          FileUtils.cp(pkg, "#{dest_dir}/")
        end
      end
      package_files.each{ |pkg| FileUtils.rm_f(pkg) }
      FileUtils.rm(fr_zip)
      FileUtils.rm(fr_bin)
    end
  end
  cyan 'Done!'
end

main(ARGV.first)
