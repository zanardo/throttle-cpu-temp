#[macro_use]
extern crate log;

use anyhow::{Context, Result};
use simplelog::{CombinedLogger, Config, LogLevelFilter, SimpleLogger};
use std::env;
use std::fs::{read_to_string, File};
use std::io::Write;
use std::path::Path;
use std::process;
use std::{thread, time};

// Sleep interval between temperature checking.
const SLEEP_TIME_MILLI: u64 = 3000;

// File where minimum supported frequency should be colected.
const MIN_FREQ_FILE: &'static str = "/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_min_freq";

// File where maximum supported frequency should be colected.
const MAX_FREQ_FILE: &'static str = "/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq";

// Step size to change cpu frequency.
const STEP_FREQ: u64 = 100000;

// Possible files where current temperature should be collected.
const POSSIBLE_TEMP_FILES: &'static [&'static str] = &[
    "/sys/class/thermal/thermal_zone0/temp",
    "/sys/class/thermal/thermal_zone1/temp",
    "/sys/class/thermal/thermal_zone2/temp",
    "/sys/class/hwmon/hwmon0/temp1_input",
    "/sys/class/hwmon/hwmon1/temp1_input",
    "/sys/class/hwmon/hwmon2/temp1_input",
    "/sys/class/hwmon/hwmon0/device/temp1_input",
    "/sys/class/hwmon/hwmon1/device/temp1_input",
    "/sys/class/hwmon/hwmon2/device/temp1_input",
];

fn parse_int_file(path: String) -> Result<u64> {
    Ok(read_to_string(path)?.trim().parse()?)
}

fn min_frequency() -> Result<u64> {
    parse_int_file(MIN_FREQ_FILE.to_string()).context("parsing MIN_FREQ_FILE")
}

fn max_frequency() -> Result<u64> {
    parse_int_file(MAX_FREQ_FILE.to_string()).context("parsing MAX_FREQ_FILE")
}

fn get_temp() -> Result<u64> {
    for file in POSSIBLE_TEMP_FILES {
        if Path::new(file).exists() {
            let temp = parse_int_file(file.to_string())?;
            return Ok(temp / 1000);
        }
    }
    error!("impossible to collect current cpu temperature!");
    process::exit(1);
}

fn set_freq(freq: u64) -> Result<()> {
    info!("setting frequency to {}", freq);
    for c in 0..num_cpus::get() {
        let path = format!("/sys/devices/system/cpu/cpu{}/cpufreq/scaling_max_freq", c);
        let mut fp = File::create(path)?;
        fp.write_all(format!("{}\n", freq).as_bytes())?;
    }
    Ok(())
}

fn main() -> Result<()> {
    CombinedLogger::init(vec![SimpleLogger::new(
        LogLevelFilter::Info,
        Config::default(),
    )])?;

    let args: Vec<String> = env::args().collect();
    if args.len() != 2 {
        error!("usage: {} <max temp>", args[0]);
        process::exit(1);
    }

    let max_temp: u64;
    match args[1].parse::<u64>() {
        Err(_) => {
            error!("invalid temperature: {}", args[1]);
            process::exit(1);
        }
        Ok(x) => max_temp = x,
    }
    info!("maximum temperature: {}", max_temp);

    info!("cpu count: {}", num_cpus::get());

    let min_freq = min_frequency()?;
    info!("minimum frequency supported: {}", min_freq);
    let max_freq = max_frequency()?;
    info!("maximum frequency supported: {}", max_freq);

    let mut cur_freq = max_freq;
    set_freq(cur_freq)?;
    loop {
        let temp = get_temp()?;
        if temp > max_temp && cur_freq > min_freq {
            cur_freq -= STEP_FREQ;
            if cur_freq < min_freq {
                cur_freq = min_freq;
            }
            set_freq(cur_freq)?;
        } else if temp < (max_temp - 5) && cur_freq < max_freq {
            cur_freq += STEP_FREQ;
            if cur_freq > max_freq {
                cur_freq = max_freq;
            }
            set_freq(cur_freq)?;
        }
        debug!("current temperature: {}", temp);
        thread::sleep(time::Duration::from_millis(SLEEP_TIME_MILLI));
    }
}
