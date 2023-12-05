use std::collections::{HashMap, HashSet, VecDeque};
use serde::{Serialize, Deserialize};
use std::error::Error;
use std::fs;
use serde_json;
use rand::Rng;
use std::net::{TcpListener, TcpStream};
use std::io::{Read, Write};
use std::thread;
use std::sync::{Arc, Mutex};
 
#[derive(Serialize, Deserialize, Clone)]
struct Set {
    data: HashSet<i32>,
}
 
impl Set {
    fn new() -> Self {
        Self { data: HashSet::new() }
    }
 
    fn insert(&mut self, value: i32) {
        self.data.insert(value);
    }
 
    fn remove(&mut self, value: &i32) {
        self.data.remove(value);
    }
 
    fn contains(&self, value: &i32) -> bool {
        self.data.contains(value)
    }
 
    fn get_all(&self) -> Vec<i32> {
        self.data.iter().cloned().collect()
    }
}
 
#[derive(Serialize, Deserialize, Clone)]
struct Stack {
    data: Vec<String>,
}
 
impl Stack {
    fn new() -> Self {
        Self { data: Vec::new() }
    }
 
    fn push(&mut self, value: String) {
        self.data.push(value);
    }
 
    fn pop(&mut self) -> Option<String> {
        self.data.pop()
    }
}
 
#[derive(Serialize, Deserialize, Clone)]
struct Queue {
    data: VecDeque<String>,
}
 
impl Queue {
    fn new() -> Self {
        Self { data: VecDeque::new() }
    }
 
    fn push_back(&mut self, value: String) {
        self.data.push_back(value);
    }
 
    fn pop_front(&mut self) -> Option<String> {
        self.data.pop_front()
    }
}
 
#[derive(Serialize, Deserialize, Clone)]
struct HashTable {
    data: Vec<Option<(String, String)>>,
    capacity: usize,
}
 
impl HashTable {
    fn new(capacity: usize) -> Self {
        Self {
            data: vec![None; capacity],
            capacity,
        }
    }
 
    fn hash(&self, key: &str) -> usize {
        key.len() % self.capacity
    }
 
    fn hash2(&self, key: &str) -> usize {
        key.chars().count() % (self.capacity - 1) + 1
    }
 
    fn insert(&mut self, key: String, value: String) -> Result<(), String> {
        if self.size() >= self.capacity / 2 {
            self.resize();
        }
 
        let mut index = self.hash(&key);
        let hash2 = self.hash2(&key);
 
        while self.data[index].is_some() {
            if let Some((existing_key, _)) = &self.data[index] {
                if existing_key == &key {
                    return Err(format!("Key '{}' already exists in the table", key));
                }
            }
            index = (index + hash2) % self.capacity;
        }
 
        self.data[index] = Some((key, value));
 
        Ok(())
    }
 
    fn remove(&mut self, key: &str) {
        let mut index = self.hash(key);
        let hash2 = self.hash2(key);
 
        while let Some((existing_key, _)) = &self.data[index] {
            if existing_key == key {
                self.data[index] = None;
                return;
            }
            index = (index + hash2) % self.capacity;
        }
    }
 
    fn get(&self, key: &str) -> Option<&String> {
        let mut index = self.hash(key);
        let hash2 = self.hash2(key);
 
        while let Some((existing_key, value)) = &self.data[index] {
            if existing_key == key {
                return Some(value);
            }
            index = (index + hash2) % self.capacity;
        }
 
        None
    }
 
    fn resize(&mut self) {
        let new_capacity = self.capacity * 2;
        let mut new_data = vec![None; new_capacity];
 
        for item in &self.data {
            if let Some((key, value)) = item {
                let mut index = self.hash(&key);
                let hash2 = self.hash2(&key);
 
                while new_data[index].is_some() {
                    index = (index + hash2) % new_capacity;
                }
 
                new_data[index] = Some((key.clone(), value.clone()));
            }
        }
 
        self.data = new_data;
        self.capacity = new_capacity;
    }
 
    fn size(&self) -> usize {
        self.data.iter().filter(|entry| entry.is_some()).count()
    }
}
 
// Структура для хранения данных о переходах
#[derive(Serialize, Deserialize, Clone)]
struct TransitionMetrics {
    url: String,
    ip_address: String,
    timestamp: String,
}

impl TransitionMetrics {
    fn new(url: &str, ip_address: &str) -> Self {
        Self {
            url: url.to_string(),
            ip_address: ip_address.to_string(),
            timestamp: chrono::Utc::now().to_rfc3339(),
        }
    }
}

 
#[derive(Serialize, Deserialize, Clone)]
struct Database {
    set: Set,
    stack: Stack,
    queue: Queue,
    hashmap: HashMap<String, HashTable>,
    transition_metrics: HashMap<String, Vec<TransitionMetrics>>, // Добавлено хранение метрик о переходах
}

impl Database {
    fn new() -> Self {
        Self {
            set: Set::new(),
            stack: Stack::new(),
            queue: Queue::new(),
            hashmap: HashMap::new(),
            transition_metrics: HashMap::new(),
        }
    }

    // Добавление записи о переходе в метрики
    fn add_transition_metrics(&mut self, short_url: &str, ip_address: &str) {
        let metrics = TransitionMetrics::new(short_url, ip_address);
        self.transition_metrics
            .entry(short_url.to_string())
            .or_insert(Vec::new())
            .push(metrics);
    }
 
    fn load(&mut self, filepath: &str) -> Result<(), Box<dyn Error>> {
        let file = fs::File::open(filepath)?;
        let reader = std::io::BufReader::new(file);
        let loaded_db: Database = serde_json::from_reader(reader)?;
        *self = loaded_db;
        Ok(())
    }
 
    fn serialize(&self) -> String {
        serde_json::to_string(self).unwrap()
    }
 
    fn execute_query(&mut self, query: &str) -> Result<String, Box<dyn Error>> {
        let parts: Vec<&str> = query.split_whitespace().collect();
        if parts.is_empty() {
            return Err("Empty query".into());
        }
 
        let command = parts[0];
 
        match command {
            "SADD" => {
                let count = parts.get(1).ok_or("No count specified")?.parse::<i32>()?;
                let mut rng = rand::thread_rng();
                for _ in 0..count {
                    let value = rng.gen_range(1..100);
                    self.set.insert(value);
                }
            }
            "SREM" => {
                let value = parts.get(1).ok_or("No value specified")?.parse::<i32>()?;
 
                if self.set.contains(&value) {
                    self.set.remove(&value);
                } else {
                    return Ok(format!("Value {} not found in the set", value));
                }
            }
            "SISMEMBER" => {
                let mut result = String::new();
                let values = self.set.get_all();
                if values.is_empty() {
                    result.push_str("Empty set");
                } else {
                    for value in values {
                        result.push_str(&format!("Value: {}\n", value));
                    }
                }
                return Ok(result);
            }
            "SPUSH" => {
                let value = parts.get(1).ok_or("No value specified")?;
                self.stack.push(value.to_string());
            }
            "SPOP" => {
                if let Some(value) = self.stack.pop() {
                    return Ok(value);
                } else {
                    return Ok(String::new());
                }
            }
            "QPUSH" => {
                let value = parts.get(1).ok_or("No value specified")?;
                self.queue.push_back(value.to_string());
            }
            "QPOP" => {
                return Ok(self.queue.pop_front().unwrap_or_default());
            }
            "HSET" => {
                let table_name = parts.get(1).ok_or("No table name specified")?.to_string();
                let key = parts.get(2).ok_or("No key specified")?.to_string();
                let value = parts.get(3).ok_or("No value specified")?.to_string();
 
                if let Some(table) = self.hashmap.get_mut(&table_name) {
                    match table.insert(key.clone(), value) {
                        Ok(()) => {
                            return Ok(format!("Key '{}' added to table '{}'", key, table_name));
                        }
                        Err(err) => {
                            return Ok(format!("Key '{}' in table '{}' is already in use: {}", key, table_name, err));
                        }
                    }
                } else {
                    let mut new_table = HashTable::new(16); // Используем стандартную емкость
                    match new_table.insert(key.clone(), value) {
                        Ok(()) => {
                            self.hashmap.insert(table_name.clone(), new_table);
                            return Ok(format!("Key '{}' added to new table '{}'", key, table_name));
                        }
                        Err(err) => {
                            return Ok(format!("Key '{}' in new table '{}' is already in use: {}", key, table_name, err));
                        }
                    }
                }
            }
            "HDEL" => {
                let table_name = parts.get(1).ok_or("No table name specified")?.to_string();
                let key = parts.get(2).ok_or("No key specified")?.to_string();
 
                if let Some(table) = self.hashmap.get_mut(&table_name) {
                    table.remove(&key);
                } else {
                    return Ok(format!("Table {} not found", table_name));
                }
            }
            "HGET" => {
                let table_name = parts.get(1).ok_or("No table name specified")?.to_string();
                let key = parts.get(2).map(|s| s.to_string());
        
                if let Some(table) = self.hashmap.get(&table_name) {
                    let mut result = String::new();
        
                    match key {
                        Some(k) => {
                            if let Some(value) = table.get(&k) {
                                return Ok(value.clone());
                            } else {
                                return Ok(format!("Key: {} not found", k));
                            }
                        }
                        None => {
                            return Err("Missing key for HGET".into());
                        }
                    }
                } else {
                    return Ok(format!("Table {} not found", table_name));
                }
            }
            _ => return Err("Unknown command".into()),
        }
 
        Ok("OK".into())
    }
}
 
fn main() {
    let args: Vec<String> = std::env::args().collect();
 
    let file = args.iter().position(|r| r == "--file")
        .map(|pos| args.get(pos + 1))
        .unwrap_or(None);
 
    let query = args.iter().position(|r| r == "--query")
        .map(|pos| args.get(pos + 1))
        .unwrap_or(None);
 
    if let (Some(file), None) = (file, query) {
        let db = Database::new();
        let db = Arc::new(Mutex::new(db)); // Оборачиваем базу данных в Mutex
        let listener = TcpListener::bind("127.0.0.1:6379").expect("Failed to bind to address");
        println!("Listening 127.0.0.1:6379");
 
        for stream in listener.incoming() {
            let db = Arc::clone(&db);
            match stream {
                Ok(stream) => {
                    thread::spawn(move || {
                        handle_client(stream, db);
                    });
                }
                Err(e) => {
                    eprintln!("Error accepting connection: {}", e);
                }
            }
        }
    } else if let (Some(file), Some(query)) = (file, query) {
        let mut db = Database::new();
 
        if let Err(e) = db.load(file) {
            eprintln!("Error loading database: {}", e);
        }
 
        match db.execute_query(query) {
            Ok(result) => println!("Result: {}", result),
            Err(e) => eprintln!("Error executing query: {}", e),
        }
 
        let serialized_db = db.serialize();
        if let Err(e) = fs::write(file, &serialized_db) {
            eprintln!("Error saving database: {}", e);
        }
    } else {
        println!("Usage: cargo run -- --file <file> --query <query>");
    }
}
 
fn handle_client(mut stream: TcpStream, db: Arc<Mutex<Database>>) {
    let mut buf = vec![0; 1024];
    while let Ok(size) = stream.read(&mut buf) {
        if size == 0 {
            // Клиент разорвал соединение, выходим из цикла
            break;
        }

        let request = String::from_utf8_lossy(&buf[..size]);
        let mut db = db.lock().unwrap(); // Захватываем мьютекс
        let response = db.execute_query(&request).unwrap_or_else(|e| e.to_string());

        // Получаем IP-адрес клиента
        let ip_address = match stream.peer_addr() {
            Ok(addr) => addr.ip().to_string(),
            Err(_) => "Unknown".to_string(),
        };

        // Записываем метрики о переходе
        if request.starts_with("GET") {
            let parts: Vec<&str> = request.split_whitespace().collect();
            if let Some(short_url) = parts.get(1) {
                db.add_transition_metrics(short_url, &ip_address);
            }
        }

        if let Err(e) = stream.write_all(response.as_bytes()) {
            eprintln!("Error writing to client: {:?}", e);
        }
    }
}