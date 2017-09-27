def findref_version
  `cat findref.go | grep 'const Version' | sed -e 's/.*=//g' | sed -e 's/"//g' | sed -e 's/\s*//g'`.chomp
end

task default: 'build'

desc 'Build the outputs from erb sources'
task :erb do
  sh "erb fr_version=#{findref_version} README.md.erb > README.md"
  sh "erb fr_version=#{findref_version} ARCHIVES.md.erb > ARCHIVES.md"
  sh "erb install.sh.erb > install.sh"
  #File.write('README.md', ERB.new(File.open('README.md.erb').read).result)
end

desc 'Build the current release'
task :release do
  ruby "build.rb #{findref_version}"
end

task build: [:release, :erb]

def alias_task(*tasks)
  tasks.each do |new_name, old_name|
    desc "Alias of #{old_name}"
    task new_name, [*Rake.application[old_name].arg_names] => [old_name]
  end
end

alias_task(
  [:b, :build]
)
