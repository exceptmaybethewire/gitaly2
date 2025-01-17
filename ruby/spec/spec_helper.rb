require_relative '../lib/gitaly_server.rb'
require_relative '../lib/gitlab/git.rb'
require_relative 'support/sentry.rb'
require 'timecop'
require 'test_repo_helper'
require 'factory_bot'

Dir[File.join(__dir__, 'support/helpers/*.rb')].each { |f| require f }

ENV['GITALY_RUBY_GIT_BIN_PATH'] ||= 'git'

RSpec.configure do |config|
  config.include FactoryBot::Syntax::Methods

  config.before(:suite) do
    FactoryBot.find_definitions
  end
end
