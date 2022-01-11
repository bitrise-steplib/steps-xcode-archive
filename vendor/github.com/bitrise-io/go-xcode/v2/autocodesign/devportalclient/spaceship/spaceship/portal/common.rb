require 'spaceship'

def preferred_error_message(ex)
  ex.preferred_error_info&.join(' ') || ex.to_s
end

def run_or_raise_preferred_error_message
  yield
rescue Spaceship::Client::UnexpectedResponse => ex
  raise preferred_error_message(ex)
end
