require 'eventmachine'

LINES_TO_COMPARE = 100

class Server
	attr_accessor :connections
	def initialize()
		@connections = []
	end
	def add_client(client)
		connections << client
		if connections.length == 2 then
			connections.each do |con|
				con.send_data("W") # Send me a packet of data!
			end
		end
		puts "Current client count: #{@connections.length}"
	end
	def remove_client(client)
		connections.delete(client)
		puts "Current client count: #{@connections.length}"
	end
	
	def dump_output()
		 #Sample: 0318 C2 13 34 0C 0A 00 ED 6A 00010010 0500
		header = "ADDR OP B  C  D  E  H  L  A  SZ-X-P-C  SP "
		client_names = @connections.collect { |con| con.client_name.center(header.length) }
		puts client_names.join(" | ")
		puts ([header]*@connections.length).join(" | ") # Put a delimeter between the headers
		LINES_TO_COMPARE.times do |line_number|
			out = @connections.collect { | con| con.lines[line_number] }
			out_str = out.join(" | ")
			if connections.collect{ |con| con.lines[line_number] }.uniq.length != 1 then
				out_str += " <-- "
			else
				out_str += " OK "
			end
			
			puts out_str
		end
	end
	
	def client_ready()
		if connections.all? { |con| con.waiting } then
			LINES_TO_COMPARE.times do |line_number|
				if connections.collect { |con| con.lines[line_number] }.uniq.length != 1 then
				#if connections[0].lines[line_number] != connections[1].lines[line_number] then
					dump_output()
					raise RuntimeError, "Discrepancy found during comparison"
				end
			end
			
			connections.each do |con|
				con.waiting = false
				con.send_data("W")
				con.lines = []
			end
		end
	end
end

$server = Server.new()

module CompareClient
	attr_accessor :waiting, :client_name, :lines
	def post_init
		puts "New connection"
		@buffer = ""
		$server.add_client(self)
		@line_counter = 0
		@client_name = ""
		@lines = []
		@waiting = false
	end
	
	def receive_data(data)
		@buffer += data
		lines = @buffer.split("\n")
		if @buffer[-1] != "\n" then
			@buffer = lines[-1]
		else
			@buffer = ""
		end
		lines.each do |line|
			elements = line.split(" ")
			if elements[0] == "IDENTIFY" then
				@client_name = elements[1..-1].join(" ")
				puts "Client identified: #{@client_name}"
			else
				@line_counter += 1
				if !@client_name then
					print "Client started sending data without first identifying"
					close_connection()
				else
					@lines << line
				end
			end
			if (@line_counter >0) && (@line_counter % LINES_TO_COMPARE == 0) then
				self.waiting = true
				$server.client_ready()
			end
			
			if (@line_counter % 100000 == 0) then
				puts "Instructions executed: #{@line_counter}"
			end
		end
	end

	def unbind
		puts "Connection lost"
		$server.remove_client(self)
	end
end

# Note that this will block current thread.
EventMachine.run {
  EventMachine.start_server "127.0.0.1", 5679, CompareClient
}