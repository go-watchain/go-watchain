Pod::Spec.new do |spec|
  spec.name         = 'Gwat'
  spec.version      = '{{.Version}}'
  spec.license      = { :type => 'GNU Lesser General Public License, Version 3.0' }
  spec.homepage     = 'https://github.com/watchain/go-watchain'
  spec.authors      = { {{range .Contributors}}
		'{{.Name}}' => '{{.Email}}',{{end}}
	}
  spec.summary      = 'iOS watchain Client'
  spec.source       = { :git => 'https://github.com/watchain/go-watchain.git', :commit => '{{.Commit}}' }

	spec.platform = :ios
  spec.ios.deployment_target  = '9.0'
	spec.ios.vendored_frameworks = 'Frameworks/Gwat.framework'

	spec.prepare_command = <<-CMD
    curl https://gwatstore.blob.core.windows.net/builds/{{.Archive}}.tar.gz | tar -xvz
    mkdir Frameworks
    mv {{.Archive}}/Gwat.framework Frameworks
    rm -rf {{.Archive}}
  CMD
end
