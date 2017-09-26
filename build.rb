#!/usr/bin/env ruby

require 'fileutils'

FR_VERSION = '0.0.1'.freeze
GO_VERSION = '1.9-alpine'.freeze

# See: https://stackoverflow.com/a/30068222/2062384 for list of valid targets
OSES_ARCHES = {
  'linux' => %w(amd64 386 arm arm64),
  'windows' => %w(amd64 386),
  'darwin' => %w(amd64 386)
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

def docker_run(os, arch)
  <<-EOS.gsub(/\s+/, ' ').gsub(/[\s\t]*\n/, ' ').strip
    docker run
    --rm
    --volume "#{Dir.pwd}:/usr/src/findref"
    --workdir "/usr/src/findref"
    --env GOOS=#{os}
    --env GOARCH=#{arch}
    golang:#{GO_VERSION} go build
  EOS
end

def main
  OSES_ARCHES.each do |os, arches|
    arches.each do |arch|
      dest_dir = "findref-bin/#{FR_VERSION}/#{os}/#{arch}"
      cyan "Building findref v#{FR_VERSION} for #{os} #{arch}..."
      system(docker_run(os, arch))
      FileUtils.mkdir_p(dest_dir)
      fr = os == 'windows' ? 'findref.exe' : 'findref'
      FileUtils.mv(fr, "#{dest_dir}/")
    end
  end
  cyan 'Done!'
end

main
