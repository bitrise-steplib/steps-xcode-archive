# Log
class Log
  @verbose = true

  class << self
     attr_accessor :verbose
  end

  def self.info(str)
    puts("\n\e[34m#{str}\e[0m")
  end

  def self.print(str)
    puts(str.to_s)
  end

  def self.success(str)
    puts("\e[32m#{str}\e[0m")
  end

  def self.warn(str)
    puts("\e[33m#{str}\e[0m")
  end

  def self.error(str)
    puts("\e[31m#{str}\e[0m")
  end

  def self.debug(str)
    puts("\e[90m#{str}\e[0m") if @verbose
  end

  def self.debug_exception(exc)
    Log.debug('Error:')
    Log.debug(exc.to_s)
    puts
    Log.debug('Stacktrace (for debugging):')
    Log.debug(exc.backtrace.join("\n").to_s)
  end

  def self.secure_value(value)
    return '' if value.empty?
    '***'
  end
end
