module ErbHelpers
  def self.glob(dir)
    Dir.glob(dir)
      .map{ |d| File.basename(d) }
  end

  def self.latest_release
    releases.first
  end

  def self.releases
    glob('findref-bin/*')
      .select{ |rel| File.directory?("findref-bin/#{rel}") }
      .map{ |rel| File.basename(rel) }
      .sort
      .reverse
  end

  def self.links(release, os)
    glob("findref-bin/#{release}/#{os}/*").map do |arch|
      "[#{arch}](https://github.com/FreedomBen/findref-bin/blob/master/#{release}/#{os}/#{arch}/#{os == 'windows' ? 'findref.exe' : 'findref'}?raw=true)"
    end
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
end
