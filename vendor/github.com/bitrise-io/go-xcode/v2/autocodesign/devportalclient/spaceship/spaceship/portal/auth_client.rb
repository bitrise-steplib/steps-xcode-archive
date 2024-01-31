require 'spaceship'

module Portal
  # AuthClient ...
  class AuthClient
    def self.login(username, password, two_factor_session = nil, team_id = nil)
      ENV['FASTLANE_SESSION'] = two_factor_session unless two_factor_session.to_s.empty?
      ENV['SPACESHIP_SKIP_2FA_UPGRADE'] = '1'

      client = Spaceship::PortalClient.login(username, password)

      if team_id.to_s.empty?
        teams = client.teams
        raise 'Your developer portal account belongs to multiple teams, please provide the team id to sign in' if teams.to_a.size > 1
      else
        client.team_id = team_id
      end

      client.store_cookie

      client.team_id
    end

    def self.restore_from_session(username, team_id)
      client = Spaceship::PortalClient.new(current_team_id: team_id)
      client.user = username
      client.load_session_from_file

      Spaceship::Portal.client = client
    end
  end
end
