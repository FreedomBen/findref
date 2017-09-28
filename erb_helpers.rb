module ErbHelpers
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
