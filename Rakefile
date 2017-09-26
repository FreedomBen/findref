task default: 'build'

desc 'Build README.md from README.md.erb'
task :readme do
  puts 'Building README.md'
  sh 'erb README.md.erb > README.md'
  #File.write('README.md', ERB.new(File.open('README.md.erb').read).result)
end

desc 'Build the current release'
task :release do
  ruby 'build.rb'
end

task build: [:readme, :release]

def alias_task(*tasks)
  tasks.each do |new_name, old_name|
    desc "Alias of #{old_name}"
    task new_name, [*Rake.application[old_name].arg_names] => [old_name]
  end
end

alias_task(
  [:b, :build]
)
