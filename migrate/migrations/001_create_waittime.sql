CREATE TABLE IF NOT EXISTS waittime (
  id BIGSERIAL PRIMARY KEY,
  location VARCHAR(10) NOT NULL,
  patients_in_waiting_room_count INTEGER NOT NULL,
  most_urgent_count INTEGER NOT NULL,
  most_urgent_waittime VARCHAR(20) NOT NULL,
  most_urgent_waittime_max INTEGER NOT NULL,
  urgent_count INTEGER NOT NULL,
  urgent_waittime VARCHAR(20) NOT NULL,
  urgent_waittime_max INTEGER NOT NULL,
  less_urgent_count INTEGER NOT NULL,
  less_urgent_waittime VARCHAR(20) NOT NULL,
  less_urgent_waittime_max INTEGER NOT NULL,
  patients_being_treated_count INTEGER NOT NULL,
  total_patients_count INTEGER NOT NULL,
  patients_waiting_transfer_count INTEGER NOT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
