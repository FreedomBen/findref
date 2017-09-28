module Helpers
  OSES_ARCHES = {
    'linux' => %w(amd64 386 arm arm64),
    'windows' => %w(amd64 386),
    'darwin' => %w(amd64 386)
  }.freeze

  def self.terminal_color_restore
    '\033[0m'
  end

  def self.terminal_colors
    {
      red: '\033[0;31m',
      blue: '\033[0;34m',
      cyan: '\033[0;36m',
      green: '\033[0;32m',
      brown: '\033[0;33m',
      black: '\033[0;30m',
      white: '\033[1;37m',
      purple: '\033[0;35m',
      yellow: '\033[1;33m',
      light_red: '\033[1;31m',
      dark_gray: '\033[1;30m',
      light_gray: '\033[0;37m',
      light_blue: '\033[1;34m',
      light_cyan: '\033[1;36m',
      light_green: '\033[1;32m',
      light_purple: '\033[1;35m'
    }
  end

  def self.bash_color_restore_line
    "declare -r color_restore='#{terminal_color_restore}'"
  end

  def self.bash_color_func(color, code, include_restore: true)
    <<-EOS.split("\n").map{|l| l.gsub(/\s{6}/, '')}.join("\n").concat("\n")
      #{include_restore ? bash_color_restore_line : ''}
      declare -r color_#{color}='#{code}'

      #{color} ()
      {
          echo -e "${color_#{color}}${1}${color_restore}\\n"
      }
    EOS
  end

  def self.bash_color_funcs
    retval = [bash_color_restore_line]
    terminal_colors.each do |color, code|
      retval.push(bash_color_func(color, code, include_restore: false))
    end
    retval.join
  end

  def self.glob(dir)
    Dir.glob(dir)
      .map{ |d| File.basename(d) }
  end

  def self.latest_release_name
    'latest'
  end

  def self.latest_release
    releases.first
  end

  def self.findref_bin_repo
    'findref-bin'
  end

  def self.releases
    glob("#{findref_bin_repo}/*")
      .select{ |rel| File.directory?("#{findref_bin_repo}/#{rel}") }
      .select{ |rel| rel != latest_release_name }
      .map{ |rel| File.basename(rel) }
      .sort
      .reverse
  end

  def self.zip_name
    'findref.zip'
  end

  def self.bin_name(os)
    os == 'windows' ? 'findref.exe' : 'findref'
  end

  def self.url(release, os, arch)
    "https://raw.githubusercontent.com/FreedomBen/#{findref_bin_repo}/master/#{release}/#{os}/#{arch}/#{zip_name}"
  end

  def self.link(release, os, arch)
    "[#{arch}](#{url(release, os, arch)})"
  end

  def self.links(release, os)
    glob("#{findref_bin_repo}/#{release}/#{os}/*").map{ |arch| link(release, os, arch) }
  end

  def self.linux_links(release)
    links(release, 'linux').join(' - ')
  end

  def self.mac_links(release)
    links(release, 'darwin').join(' - ')
  end

  def self.windows_links(release)
    links(release, 'windows').join(' - ')
  end

  def self.release_line(release)
    "| #{release} | #{linux_links(release)} | #{mac_links(release)} | #{windows_links(release)} |"
  end

  def self.release_line_latest
    release_line(latest_release_name)
  end

  def self.release_table_header
    <<-EOS.split("\n").map(&:strip).join("\n").concat("\n")
      | Version | Linux | macOS | Windows |
      |:-------:|:-----:|:-----:|:-------:|
    EOS
  end

  def self.release_table(release_lines = nil)
    release_lines ||= releases.map{ |r| release_line(r) }
    release_lines = Array(release_lines)
    "#{release_table_header}#{release_lines.join("\n")}"
  end
end
